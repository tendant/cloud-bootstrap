package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gopkg.in/yaml.v3"
)

// Bootstrapper handles AWS resource provisioning
type Bootstrapper struct {
	awsConfig aws.Config
	ctx       context.Context
}

// NewBootstrapper creates a new Bootstrapper instance
func NewBootstrapper(region string) (*Bootstrapper, error) {
	ctx := context.TODO()

	// Load AWS configuration with explicit region and retry options
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithRetryMaxAttempts(3),
		config.WithRetryMode(aws.RetryModeStandard),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS config: %w", err)
	}

	// Validate that AWS credentials are available
	_, err = awsConfig.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	return &Bootstrapper{
		awsConfig: awsConfig,
		ctx:       ctx,
	}, nil
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &config, nil
}

// ProvisionResources provisions all resources defined in the configuration
func (b *Bootstrapper) ProvisionResources(config *Config) error {
	// Create S3 buckets
	if err := b.CreateS3Buckets(config.S3Buckets); err != nil {
		return fmt.Errorf("failed to create S3 buckets: %w", err)
	}

	// Create ECR repositories
	if err := b.CreateECRRepositories(config.ECRRepositories); err != nil {
		return fmt.Errorf("failed to create ECR repositories: %w", err)
	}

	// Create IAM users and policies
	if err := b.CreateIAMUsersAndPolicies(config.IAMUsers); err != nil {
		return fmt.Errorf("failed to create IAM users and policies: %w", err)
	}

	// Manage RDS instances
	if err := b.ManageRDSInstances(config.RDSInstances); err != nil {
		return fmt.Errorf("failed to manage RDS instances: %w", err)
	}

	return nil
}

// CreateS3Buckets creates S3 buckets based on the configuration
func (b *Bootstrapper) CreateS3Buckets(buckets []S3Bucket) error {
	s3Client := s3.NewFromConfig(b.awsConfig)

	for _, bucket := range buckets {
		fmt.Printf("Ensuring S3 bucket: %s\n", bucket.Name)

		// Check if bucket exists
		_, err := s3Client.HeadBucket(b.ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucket.Name),
		})

		if err != nil {
			// Bucket doesn't exist, create it
			createBucketInput := &s3.CreateBucketInput{
				Bucket: aws.String(bucket.Name),
			}

			// Add location constraint if not in us-east-1
			if b.awsConfig.Region != "us-east-1" {
				createBucketInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
					LocationConstraint: types.BucketLocationConstraint(b.awsConfig.Region),
				}
			}

			_, err = s3Client.CreateBucket(b.ctx, createBucketInput)
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket.Name, err)
			}
			fmt.Printf("✅ Created bucket: %s\n", bucket.Name)
		} else {
			fmt.Printf("✅ Bucket %s already exists\n", bucket.Name)
		}

		// Configure versioning
		if bucket.Versioning == "enabled" {
			_, err = s3Client.PutBucketVersioning(b.ctx, &s3.PutBucketVersioningInput{
				Bucket: aws.String(bucket.Name),
				VersioningConfiguration: &types.VersioningConfiguration{
					Status: types.BucketVersioningStatusEnabled,
				},
			})
			if err != nil {
				fmt.Printf("⚠️ Warning: failed to enable versioning for bucket %s: %v\n", bucket.Name, err)
			} else {
				fmt.Printf("✅ Enabled versioning for bucket: %s\n", bucket.Name)
			}
		}

		// Configure encryption
		if bucket.Encryption != "" {
			_, err = s3Client.PutBucketEncryption(b.ctx, &s3.PutBucketEncryptionInput{
				Bucket: aws.String(bucket.Name),
				ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
					Rules: []types.ServerSideEncryptionRule{
						{
							ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
								SSEAlgorithm: types.ServerSideEncryptionAes256,
							},
						},
					},
				},
			})
			if err != nil {
				fmt.Printf("⚠️ Warning: failed to configure encryption for bucket %s: %v\n", bucket.Name, err)
			} else {
				fmt.Printf("✅ Configured encryption for bucket: %s\n", bucket.Name)
			}
		}

		// Configure CORS
		if bucket.CORS != nil {
			corsRules := []types.CORSRule{
				{
					AllowedOrigins: bucket.CORS.AllowedOrigins,
					AllowedMethods: convertToMethodsEnum(bucket.CORS.AllowedMethods),
					AllowedHeaders: bucket.CORS.AllowedHeaders,
					ExposeHeaders:  bucket.CORS.ExposeHeaders,
					MaxAgeSeconds:  aws.Int32(int32(bucket.CORS.MaxAgeSeconds)),
				},
			}

			_, err = s3Client.PutBucketCors(b.ctx, &s3.PutBucketCorsInput{
				Bucket: aws.String(bucket.Name),
				CORSConfiguration: &types.CORSConfiguration{
					CORSRules: corsRules,
				},
			})
			if err != nil {
				fmt.Printf("⚠️ Warning: failed to configure CORS for bucket %s: %v\n", bucket.Name, err)
			} else {
				fmt.Printf("✅ Configured CORS for bucket: %s\n", bucket.Name)
			}
		}

		// Configure bucket policy
		if bucket.Policy != "" {
			_, err = s3Client.PutBucketPolicy(b.ctx, &s3.PutBucketPolicyInput{
				Bucket: aws.String(bucket.Name),
				Policy: aws.String(bucket.Policy),
			})
			if err != nil {
				fmt.Printf("⚠️ Warning: failed to set policy for bucket %s: %v\n", bucket.Name, err)
			} else {
				fmt.Printf("✅ Set policy for bucket: %s\n", bucket.Name)
			}
		}
	}

	return nil
}

// CreateECRRepositories creates ECR repositories based on the configuration
func (b *Bootstrapper) CreateECRRepositories(repositories []ECRRepository) error {
	ecrClient := ecr.NewFromConfig(b.awsConfig)

	for _, repo := range repositories {
		fmt.Printf("Ensuring ECR repository: %s\n", repo.Name)

		// Check if repository exists
		_, err := ecrClient.DescribeRepositories(b.ctx, &ecr.DescribeRepositoriesInput{
			RepositoryNames: []string{repo.Name},
		})

		if err != nil {
			// Repository doesn't exist, create it
			_, err = ecrClient.CreateRepository(b.ctx, &ecr.CreateRepositoryInput{
				RepositoryName: aws.String(repo.Name),
			})
			if err != nil {
				return fmt.Errorf("failed to create ECR repository %s: %w", repo.Name, err)
			}
			fmt.Printf("✅ Created ECR repository: %s\n", repo.Name)
		} else {
			fmt.Printf("✅ ECR repository %s already exists\n", repo.Name)
		}

		// Set lifecycle policy if provided
		if repo.LifecyclePolicy != "" {
			_, err = ecrClient.PutLifecyclePolicy(b.ctx, &ecr.PutLifecyclePolicyInput{
				RepositoryName:      aws.String(repo.Name),
				LifecyclePolicyText: aws.String(repo.LifecyclePolicy),
			})
			if err != nil {
				fmt.Printf("⚠️ Warning: failed to set lifecycle policy for ECR repository %s: %v\n", repo.Name, err)
			} else {
				fmt.Printf("✅ Set lifecycle policy for ECR repository: %s\n", repo.Name)
			}
		}
	}

	return nil
}

// CreateIAMUsersAndPolicies creates IAM users and policies based on the configuration
func (b *Bootstrapper) CreateIAMUsersAndPolicies(users []IAMUser) error {
	iamClient := iam.NewFromConfig(b.awsConfig)

	for _, user := range users {
		fmt.Printf("Ensuring IAM user: %s\n", user.Name)

		// Check if user exists
		_, err := iamClient.GetUser(b.ctx, &iam.GetUserInput{
			UserName: aws.String(user.Name),
		})

		if err != nil {
			// User doesn't exist, create it
			_, err = iamClient.CreateUser(b.ctx, &iam.CreateUserInput{
				UserName: aws.String(user.Name),
			})
			if err != nil {
				return fmt.Errorf("failed to create IAM user %s: %w", user.Name, err)
			}
			fmt.Printf("✅ Created IAM user: %s\n", user.Name)
		} else {
			fmt.Printf("✅ IAM user %s already exists\n", user.Name)
		}

		// Create and attach policies
		for _, policy := range user.Policies {
			policyArn, err := b.createIAMPolicy(iamClient, user.Name, policy)
			if err != nil {
				return err
			}

			// Attach policy to user
			_, err = iamClient.AttachUserPolicy(b.ctx, &iam.AttachUserPolicyInput{
				UserName:  aws.String(user.Name),
				PolicyArn: aws.String(policyArn),
			})
			if err != nil {
				// Check if policy is already attached (which is fine)
				if strings.Contains(err.Error(), "EntityAlreadyExists") {
					fmt.Printf("✅ Policy %s already attached to user %s\n", policy.Name, user.Name)
				} else {
					fmt.Printf("⚠️ Warning: failed to attach policy %s to user %s: %v\n", policy.Name, user.Name, err)
				}
			} else {
				fmt.Printf("✅ Attached policy %s to user %s\n", policy.Name, user.Name)
			}
		}
	}

	return nil
}

// createIAMPolicy creates or updates an IAM policy and returns its ARN
func (b *Bootstrapper) createIAMPolicy(iamClient *iam.Client, userName string, policy IAMPolicy) (string, error) {
	// Create policy name with user prefix to avoid conflicts
	fullPolicyName := fmt.Sprintf("%s-%s", userName, policy.Name)

	// Check if policy exists
	listPoliciesOutput, err := iamClient.ListPolicies(b.ctx, &iam.ListPoliciesInput{
		Scope: "Local",
	})
	if err != nil {
		return "", fmt.Errorf("failed to list IAM policies: %w", err)
	}

	for _, p := range listPoliciesOutput.Policies {
		if *p.PolicyName == fullPolicyName {
			fmt.Printf("✅ IAM policy %s already exists, updating policy document\n", fullPolicyName)
			
			// Get the policy version to update
			policyArn := *p.Arn
			
			// Create a new version of the policy (this effectively updates it)
			_, err := iamClient.CreatePolicyVersion(b.ctx, &iam.CreatePolicyVersionInput{
				PolicyArn:      aws.String(policyArn),
				PolicyDocument: aws.String(policy.PolicyDocument),
				SetAsDefault:   true,
			})
			if err != nil {
				return "", fmt.Errorf("failed to update IAM policy %s: %w", fullPolicyName, err)
			}
			
			fmt.Printf("✅ Updated IAM policy: %s\n", fullPolicyName)
			return policyArn, nil
		}
	}

	// Policy doesn't exist, create it
	createPolicyOutput, err := iamClient.CreatePolicy(b.ctx, &iam.CreatePolicyInput{
		PolicyName:     aws.String(fullPolicyName),
		Description:    aws.String(policy.Description),
		PolicyDocument: aws.String(policy.PolicyDocument),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create IAM policy %s: %w", fullPolicyName, err)
	}

	fmt.Printf("✅ Created IAM policy: %s\n", fullPolicyName)
	return *createPolicyOutput.Policy.Arn, nil
}

// Helper function to convert methods to uppercase
func convertToMethodsEnum(methods []string) []string {
	result := make([]string, len(methods))
	for i, method := range methods {
		result[i] = strings.ToUpper(method)
	}
	return result
}

// ManageRDSInstances creates or modifies RDS instances based on the configuration
func (b *Bootstrapper) ManageRDSInstances(instances []RDSInstance) error {
	// Skip if no instances are defined
	if len(instances) == 0 {
		return nil
	}

	// Create RDS client
	rdsClient := rds.NewFromConfig(b.awsConfig)

	for _, instance := range instances {
		fmt.Printf("Ensuring RDS instance: %s\n", instance.Identifier)

		// Check if the instance exists
		describeInput := &rds.DescribeDBInstancesInput{
			DBInstanceIdentifier: aws.String(instance.Identifier),
		}

		describeOutput, err := rdsClient.DescribeDBInstances(b.ctx, describeInput)
		
		if err != nil {
			// Instance doesn't exist, create it
			if strings.Contains(err.Error(), "DBInstanceNotFound") {
				// Create new RDS instance
				fmt.Printf("Creating new RDS instance: %s\n", instance.Identifier)
				
				// Set up creation parameters
				createInput := &rds.CreateDBInstanceInput{
					DBInstanceIdentifier:    aws.String(instance.Identifier),
					Engine:                  aws.String(instance.Engine),
					DBInstanceClass:         aws.String(instance.InstanceClass),
					AllocatedStorage:        aws.Int32(int32(instance.AllocatedStorage)),
					DBName:                  aws.String(instance.DBName),
				}

				// Add optional parameters if provided
				if instance.EngineVersion != "" {
					createInput.EngineVersion = aws.String(instance.EngineVersion)
				}
				
				if instance.StorageType != "" {
					createInput.StorageType = aws.String(instance.StorageType)
				}
				
				if instance.MasterUsername != "" {
					createInput.MasterUsername = aws.String(instance.MasterUsername)
				}
				
				if instance.MasterPassword != "" {
					createInput.MasterUserPassword = aws.String(instance.MasterPassword)
				}
				
				createInput.PubliclyAccessible = aws.Bool(instance.PubliclyAccessible)
				
				if instance.BackupRetentionPeriod > 0 {
					createInput.BackupRetentionPeriod = aws.Int32(int32(instance.BackupRetentionPeriod))
				}
				
				createInput.MultiAZ = aws.Bool(instance.MultiAZ)

				// Handle final snapshot setting
				// For AWS SDK compatibility, we need to adapt our configuration to the actual API fields
				// SkipFinalSnapshot is handled differently in the AWS SDK
				if instance.SkipFinalSnapshot {
					// When skipping final snapshot, no need to specify a snapshot ID
					// This is the equivalent of setting SkipFinalSnapshot to true
				} else {
					// When not skipping, we need to provide a snapshot ID
					// The AWS SDK requires this field when not skipping the final snapshot
					createInput.DBName = aws.String(fmt.Sprintf("%s-final-snapshot", instance.Identifier))
				}

				// Create the instance
				_, err = rdsClient.CreateDBInstance(b.ctx, createInput)
				if err != nil {
					return fmt.Errorf("failed to create RDS instance %s: %w", instance.Identifier, err)
				}
				
				fmt.Printf("✅ Created RDS instance: %s\n", instance.Identifier)
			} else {
				// Some other error occurred
				return fmt.Errorf("error checking RDS instance %s: %w", instance.Identifier, err)
			}
		} else {
			// Instance exists, check if we need to modify it
			if len(describeOutput.DBInstances) > 0 {
				existingInstance := describeOutput.DBInstances[0]
				
				// Get current storage size (safely handle nil pointer)
				var currentStorage int32
				if existingInstance.AllocatedStorage != nil {
					currentStorage = *existingInstance.AllocatedStorage
				}
				
				// Check if storage size needs to be updated
				if currentStorage != int32(instance.AllocatedStorage) {
					fmt.Printf("Modifying storage size for RDS instance %s from %d GB to %d GB\n", 
						instance.Identifier, currentStorage, instance.AllocatedStorage)
					
					// Check if the instance is in a modifiable state (safely handle nil pointer)
					var instanceStatus string
					if existingInstance.DBInstanceStatus != nil {
						instanceStatus = *existingInstance.DBInstanceStatus
					}
					
					if instanceStatus != "available" {
						fmt.Printf("⚠️ Warning: Cannot modify RDS instance %s because it is in %s state. Must be 'available'.\n", 
							instance.Identifier, instanceStatus)
						continue
					}
					
					// Modify the instance storage
					modifyInput := &rds.ModifyDBInstanceInput{
						DBInstanceIdentifier: aws.String(instance.Identifier),
						AllocatedStorage:     aws.Int32(int32(instance.AllocatedStorage)),
						ApplyImmediately:     aws.Bool(true),
					}
					
					_, err = rdsClient.ModifyDBInstance(b.ctx, modifyInput)
					if err != nil {
						fmt.Printf("⚠️ Warning: failed to modify storage for RDS instance %s: %v\n", instance.Identifier, err)
					} else {
						fmt.Printf("✅ Modified storage for RDS instance %s to %d GB\n", 
							instance.Identifier, instance.AllocatedStorage)
						fmt.Printf("   Note: Storage modification is in progress and may take several minutes to complete\n")
					}
				} else {
					fmt.Printf("✅ RDS instance %s already exists with correct storage size (%d GB)\n", 
						instance.Identifier, currentStorage)
				}
				
				// Check if instance class needs to be updated (safely handle nil pointer)
				var currentInstanceClass string
				if existingInstance.DBInstanceClass != nil {
					currentInstanceClass = *existingInstance.DBInstanceClass
				}
				
				if currentInstanceClass != "" && currentInstanceClass != instance.InstanceClass {
					fmt.Printf("Instance class change detected (%s -> %s), but not implemented in this version\n", 
						currentInstanceClass, instance.InstanceClass)
				}
				
				// Check if engine version needs to be updated (safely handle nil pointer)
				var currentEngineVersion string
				if existingInstance.EngineVersion != nil {
					currentEngineVersion = *existingInstance.EngineVersion
				}
				
				if instance.EngineVersion != "" && currentEngineVersion != "" && 
				   currentEngineVersion != instance.EngineVersion {
					fmt.Printf("Engine version change detected (%s -> %s), but not implemented in this version\n", 
						currentEngineVersion, instance.EngineVersion)
				}
			}
		}
	}
	
	return nil
}

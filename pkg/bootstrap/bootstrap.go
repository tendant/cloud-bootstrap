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
				return fmt.Errorf("failed to enable versioning for bucket %s: %w", bucket.Name, err)
			}
			fmt.Printf("✅ Enabled versioning for bucket: %s\n", bucket.Name)
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
				return fmt.Errorf("failed to configure encryption for bucket %s: %w", bucket.Name, err)
			}
			fmt.Printf("✅ Configured encryption for bucket: %s\n", bucket.Name)
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
				return fmt.Errorf("failed to configure CORS for bucket %s: %w", bucket.Name, err)
			}
			fmt.Printf("✅ Configured CORS for bucket: %s\n", bucket.Name)
		}

		// Configure bucket policy
		if bucket.Policy != "" {
			_, err = s3Client.PutBucketPolicy(b.ctx, &s3.PutBucketPolicyInput{
				Bucket: aws.String(bucket.Name),
				Policy: aws.String(bucket.Policy),
			})
			if err != nil {
				return fmt.Errorf("failed to set policy for bucket %s: %w", bucket.Name, err)
			}
			fmt.Printf("✅ Set policy for bucket: %s\n", bucket.Name)
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
				return fmt.Errorf("failed to set lifecycle policy for ECR repository %s: %w", repo.Name, err)
			}
			fmt.Printf("✅ Set lifecycle policy for ECR repository: %s\n", repo.Name)
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
				return fmt.Errorf("failed to attach policy %s to user %s: %w", policy.Name, user.Name, err)
			}
			fmt.Printf("✅ Attached policy %s to user %s\n", policy.Name, user.Name)
		}
	}

	return nil
}

// createIAMPolicy creates an IAM policy and returns its ARN
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
			fmt.Printf("✅ IAM policy %s already exists\n", fullPolicyName)
			return *p.Arn, nil
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

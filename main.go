package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/tendant/cloud-bootstrap/pkg/bootstrap"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "aws-resources.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Run in dry-run mode without making changes")
	checkCreds := flag.Bool("check-creds", false, "Only check AWS credentials and exit")
	flag.Parse()

	// Load configuration
	config, err := bootstrap.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Check AWS credentials first
	fmt.Println("Checking AWS credentials...")
	fmt.Println(bootstrap.GetAWSProfileInfo())

	arn, err := bootstrap.CheckAWSCredentials(context.Background(), config.Region)
	if err != nil {
		log.Fatalf("AWS credential check failed: %v", err)
	}

	fmt.Printf("✅ AWS credentials validated. Authenticated as: %s\n\n", arn)

	// If only checking credentials, exit now
	if *checkCreds {
		fmt.Println("Credential check completed successfully.")
		return
	}

	// Check if dry run mode is enabled
	if *dryRun {
		fmt.Println("Running in dry-run mode. No changes will be made.")
		printPlannedChanges(config)
		return
	}

	// Initialize bootstrapper
	bootstrapper, err := bootstrap.NewBootstrapper(config.Region)
	if err != nil {
		log.Fatalf("Failed to initialize bootstrapper: %v\n\nPlease check your AWS credentials and region configuration.\nMake sure you have valid credentials in ~/.aws/credentials or environment variables.\n", err)
	}

	// Provision resources
	if err := bootstrapper.ProvisionResources(config); err != nil {
		log.Fatalf("Failed to provision resources: %v", err)
	}

	fmt.Println("✅ All resources configured successfully.")
}

// printPlannedChanges prints what would be done in dry-run mode
func printPlannedChanges(config *bootstrap.Config) {
	fmt.Println("The following resources would be provisioned:")

	// Print S3 buckets
	if len(config.S3Buckets) > 0 {
		fmt.Println("\nS3 Buckets:")
		for _, bucket := range config.S3Buckets {
			fmt.Printf("  - %s\n", bucket.Name)
			if bucket.Versioning == "enabled" {
				fmt.Println("    - Versioning: enabled")
			}
			if bucket.Encryption != "" {
				fmt.Printf("    - Encryption: %s\n", bucket.Encryption)
			}
			if bucket.CORS != nil {
				fmt.Println("    - CORS configuration would be applied")
			}
			if bucket.Policy != "" {
				fmt.Println("    - Bucket policy would be applied")
			}
		}
	}

	// Print ECR repositories
	if len(config.ECRRepositories) > 0 {
		fmt.Println("\nECR Repositories:")
		for _, repo := range config.ECRRepositories {
			fmt.Printf("  - %s\n", repo.Name)
			if repo.LifecyclePolicy != "" {
				fmt.Println("    - Lifecycle policy would be applied")
			}
		}
	}

	// Print IAM users
	if len(config.IAMUsers) > 0 {
		fmt.Println("\nIAM Users:")
		for _, user := range config.IAMUsers {
			fmt.Printf("  - %s\n", user.Name)
			if len(user.Policies) > 0 {
				fmt.Println("    Policies:")
				for _, policy := range user.Policies {
					fmt.Printf("    - %s: %s\n", policy.Name, policy.Description)
				}
			}
		}
	}
}

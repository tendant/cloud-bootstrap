package bootstrap

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// CheckAWSCredentials validates AWS credentials and returns information about the authenticated user
func CheckAWSCredentials(ctx context.Context, region string) (string, error) {
	// Load AWS configuration with environment variables prioritized
	// The AWS SDK's default credential provider chain checks environment variables first,
	// then falls back to other sources like instance role
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create STS client
	stsClient := sts.NewFromConfig(cfg)

	// Get caller identity to verify credentials
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		// Check common credential sources
		credSources := []string{
			"AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables",
			"~/.aws/credentials file",
			"EC2 instance profile or ECS task role",
		}

		var credInfo string
		for _, source := range credSources {
			credInfo += fmt.Sprintf("  - %s\n", source)
		}

		return "", fmt.Errorf("failed to validate AWS credentials: %w\n\nCredentials can be configured via:\n%s", err, credInfo)
	}

	return aws.ToString(identity.Arn), nil
}

// GetAWSProfileInfo returns information about the current AWS profile
func GetAWSProfileInfo() string {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			region = "unknown"
		}
	}

	return fmt.Sprintf("AWS Profile: %s, Region: %s", profile, region)
}

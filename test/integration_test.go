package test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// TestS3BucketIntegration performs an integration test with AWS
// Note: This test requires valid AWS credentials and will create actual resources
// Set the TEST_INTEGRATION environment variable to run this test
func TestS3BucketIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set TEST_INTEGRATION=1 to run")
	}

	// Use a unique test bucket name
	testBucketName := "test-bucket-" + os.Getenv("USER") + "-" + os.Getenv("TEST_RUN_ID")

	// Initialize AWS config
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		t.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Create test bucket
	_, err = s3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: &testBucketName,
	})
	if err != nil {
		t.Fatalf("Failed to create test bucket: %v", err)
	}

	// Clean up after test
	defer func() {
		_, err = s3Client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{
			Bucket: &testBucketName,
		})
		if err != nil {
			t.Logf("Warning: Failed to delete test bucket: %v", err)
		}
	}()

	// Verify bucket exists
	_, err = s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: &testBucketName,
	})
	if err != nil {
		t.Fatalf("Test bucket does not exist: %v", err)
	}

	t.Logf("Successfully created and verified test bucket: %s", testBucketName)
}

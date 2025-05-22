package main

import (
	"os"
	"testing"

	"github.com/tendant/cloud-bootstrap/pkg/bootstrap"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	testConfig := `
region: us-west-2
s3_buckets:
  - name: test-bucket
    versioning: enabled
    encryption: AES256
ecr_repositories:
  - name: test-repo
iam_users:
  - name: test-user
    policies:
      - name: test-policy
        description: "Test policy"
        policy_document: >
          {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": ["s3:GetObject"],
                "Resource": ["arn:aws:s3:::test-bucket/*"]
              }
            ]
          }
`
	tmpfile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testConfig)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config
	config, err := bootstrap.LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the config was loaded correctly
	if config.Region != "us-west-2" {
		t.Errorf("Expected region to be us-west-2, got %s", config.Region)
	}
	if len(config.S3Buckets) != 1 || config.S3Buckets[0].Name != "test-bucket" {
		t.Errorf("S3 bucket config not loaded correctly")
	}
	if len(config.ECRRepositories) != 1 || config.ECRRepositories[0].Name != "test-repo" {
		t.Errorf("ECR repository config not loaded correctly")
	}
	if len(config.IAMUsers) != 1 || config.IAMUsers[0].Name != "test-user" {
		t.Errorf("IAM user config not loaded correctly")
	}
}

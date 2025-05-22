# Cloud Bootstrap for Infrastructure

This project provides a YAML-based approach to define and provision AWS resources including S3 buckets, ECR repositories, and IAM users with policies.

## Overview

The AWS bootstrap tool uses a YAML configuration file to define AWS resources and a Go application to provision these resources in your AWS account. This approach provides several benefits:

- **Infrastructure as Code**: All AWS resources are defined in a YAML file, making it easy to version control and review changes
- **Idempotent Operations**: Resources are only created if they don't already exist
- **Consistent Configuration**: Standardized configuration for all resources

## Prerequisites

- Go 1.20 or later
- AWS CLI configured with appropriate credentials
- AWS SDK for Go v2

## Getting Started

1. Clone this repository
2. Configure your AWS resources in `aws-resources.yaml`
3. Run the bootstrap application:

```bash
go run main.go
```

## Configuration File

The `aws-resources.yaml` file defines all AWS resources to be provisioned. The configuration uses raw JSON embedded directly in the YAML file for policies and other complex configurations. Here's an overview of the configuration structure:

```yaml
# AWS Resources Configuration
region: us-west-2

# S3 Buckets Configuration
s3_buckets:
  - name: my-bucket-name
    versioning: enabled
    encryption: AES256
    cors:
      allowed_origins:
        - "https://example.com"
      allowed_methods:
        - "GET"
        - "PUT"
      allowed_headers:
        - "*"
      expose_headers:
        - "ETag"
        - "Content-Length"
      max_age_seconds: 3000
    policy: >
      {
        "Version": "2012-10-17",
        "Statement": [
          {
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::my-bucket-name/*"
          }
        ]
      }

# ECR Repositories Configuration
ecr_repositories:
  - name: my-service-api
    lifecycle_policy: >
      {
        "rules": [
          {
            "rulePriority": 1,
            "description": "Keep last 10 images",
            "selection": {
              "tagStatus": "any",
              "countType": "imageCountMoreThan",
              "countNumber": 10
            },
            "action": {
              "type": "expire"
            }
          }
        ]
      }

# IAM Users Configuration
iam_users:
  - name: s3-access-user
    policies:
      - name: s3-bucket-access
        description: "Policy to allow access to S3 buckets"
        policy_document: >
          {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": [
                  "s3:GetObject",
                  "s3:PutObject",
                  "s3:ListBucket"
                ],
                "Resource": [
                  "arn:aws:s3:::my-bucket-name/*"
                ]
              }
            ]
          }
```

## Features

### S3 Bucket Creation

The tool can create S3 buckets with the following configurations:
- Versioning
- Server-side encryption
- CORS configuration
- Bucket policies

### S3 Bucket CORS Configuration

CORS (Cross-Origin Resource Sharing) can be configured for each S3 bucket with:
- Allowed origins
- Allowed methods
- Allowed headers
- Expose headers
- Max age seconds

### S3 Bucket Policies

Bucket policies are defined using raw JSON directly in the YAML file:

```yaml
policy: >
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Principal": "*",
        "Action": "s3:GetObject",
        "Resource": "arn:aws:s3:::my-bucket-name/*"
      }
    ]
  }
```

### ECR Repository Creation

The tool creates ECR repositories with lifecycle policies to manage image retention. The lifecycle policies are defined using raw JSON:

```yaml
lifecycle_policy: >
  {
    "rules": [
      {
        "rulePriority": 1,
        "description": "Keep last 10 images",
        "selection": {
          "tagStatus": "any",
          "countType": "imageCountMoreThan",
          "countNumber": 10
        },
        "action": {
          "type": "expire"
        }
      }
    ]
  }
```

### IAM User Creation

The tool creates IAM users and attaches policies to them. Policies are defined using raw JSON directly in the YAML file:

```yaml
policy_document: >
  {
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "s3:GetObject",
          "s3:PutObject",
          "s3:ListBucket"
        ],
        "Resource": [
          "arn:aws:s3:::my-bucket-name/*"
        ]
      }
    ]
  }
```

## Example Usage

1. Define your AWS resources in `aws-resources.yaml`
2. Run the bootstrap application:

```bash
go run main.go
```

3. The application will create all resources defined in the configuration file

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

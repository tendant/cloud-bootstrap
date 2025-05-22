#!/bin/bash
set -euo pipefail

# Set AWS profile for testing
export AWS_PROFILE=test-profile

# Run the bootstrap application with verbose output
go run main.go -verbose

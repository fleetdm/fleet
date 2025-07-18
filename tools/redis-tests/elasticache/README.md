# ElastiCache IAM Authentication Test Tools

This directory contains tools for testing Fleet's ElastiCache IAM authentication implementation.

## Prerequisites

- AWS CLI configured with appropriate credentials (via `AWS_PROFILE` or `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`)

## Usage

### 1. Deploy Test Environment

```bash
# Set AWS credentials and region
export AWS_PROFILE=your-profile  # or use AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
export AWS_REGION=us-east-2

# Deploy the test environment
./deploy-test-env.sh
```

This creates:
- VPC with subnets and security groups
- EC2 instance for running tests
- ElastiCache Serverless instance with IAM auth
- ElastiCache Standalone replication group with IAM auth
- IAM users and roles for authentication

### 2. Clean Up

```bash
./cleanup-test-env.sh
```

This will destroy the AWS resources.
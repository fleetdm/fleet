# RDS IAM Authentication Test Tools

This directory contains test tools for validating Fleet's AWS RDS IAM authentication implementation.

## Prerequisites

- AWS CLI configured with appropriate credentials (via `AWS_PROFILE` or `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`)

## Usage

### Deploy Test Environment

```bash
# Set AWS credentials and region
export AWS_PROFILE=your-profile  # or use AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
export AWS_REGION=us-east-2

./deploy-test-env.sh
```

This will:
- Create all RDS instances with IAM authentication enabled
- Create an EC2 instance with necessary IAM permissions
- Create database users for IAM authentication
- Output connection information

### Clean Up

```bash
./cleanup-test-env.sh
```

This will destroy the AWS resources.
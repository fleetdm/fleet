#!/bin/bash
set -euo pipefail

# Configuration - use environment variables or defaults
VPC_CIDR="10.0.0.0/16"
SUBNET_CIDR="10.0.1.0/24"
STACK_NAME="${STACK_NAME:-fleet-elasticache-iam-test}"
KEY_NAME="${KEY_NAME:-fleet-test-key-$(date +%s)}"
INSTANCE_TYPE="${INSTANCE_TYPE:-t3.micro}"


echo "🚀 Deploying test environment for ElastiCache IAM authentication"

echo "📦 Creating EC2 key pair..."
aws ec2 create-key-pair \
    --key-name "$KEY_NAME" \
    --query 'KeyMaterial' \
    --output text > "${KEY_NAME}.pem"
chmod 600 "${KEY_NAME}.pem"
echo "✅ Key pair created: ${KEY_NAME}.pem"

echo "🏗️  Deploying CloudFormation stack..."
aws cloudformation create-stack \
    --stack-name "$STACK_NAME" \
    --template-body file://cf-template.yaml \
    --parameters ParameterKey=KeyName,ParameterValue="$KEY_NAME" ParameterKey=InstanceType,ParameterValue="$INSTANCE_TYPE" \
    --capabilities CAPABILITY_NAMED_IAM

echo "⏳ Waiting for stack creation to complete (this may take 5-10 minutes)..."
aws cloudformation wait stack-create-complete \
    --stack-name "$STACK_NAME"

echo "✅ Stack created successfully!"

# Get outputs
CACHE_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`ServerlessCacheEndpoint`].OutputValue' \
    --output text)

EC2_IP=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`EC2InstancePublicIP`].OutputValue' \
    --output text)

EC2_ID=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`EC2InstanceId`].OutputValue' \
    --output text)

CACHE_USER=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`CacheUser`].OutputValue' \
    --output text)

echo "📋 Stack Outputs:"
echo "  ElastiCache Endpoint: $CACHE_ENDPOINT"
echo "  EC2 Public IP: $EC2_IP"
echo "  EC2 Instance ID: $EC2_ID"
echo "  Cache User: $CACHE_USER"
echo "  SSH Key: ${KEY_NAME}.pem"

# Wait for instance to be ready
echo "⏳ Waiting for EC2 instance to be ready..."
aws ec2 wait instance-status-ok \
    --instance-ids "$EC2_ID"

# Save connection info
cat > test-env-info.txt << EOF
CACHE_ENDPOINT=$CACHE_ENDPOINT
EC2_IP=$EC2_IP
EC2_ID=$EC2_ID
CACHE_USER=$CACHE_USER
SSH_KEY=${KEY_NAME}.pem
EOF

echo "✅ Test environment deployed successfully!"
echo ""
echo "📝 Next steps:"
echo "1. Cross-compile the test binary:"
echo "   GOOS=linux GOARCH=amd64 go build -o iamconnect"
echo ""
echo "2. Copy the binary to EC2:"
echo "   scp -i ${KEY_NAME}.pem iamconnect ec2-user@$EC2_IP:~/"
echo ""
echo "3. SSH to the instance:"
echo "   ssh -i ${KEY_NAME}.pem ec2-user@$EC2_IP"
echo ""
echo "4. Run the test:"
echo "   ./iamconnect -addr $CACHE_ENDPOINT:6379 -user $CACHE_USER"
echo ""
echo "To clean up when done:"
echo "   ./cleanup-test-env.sh"
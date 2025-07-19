#!/bin/bash
set -euo pipefail

# Configuration - use environment variables or defaults
STACK_NAME="${STACK_NAME:-fleet-mysql-iam-test}"
KEY_NAME="${KEY_NAME:-fleet-mysql-test-key-$(date +%s)}"
INSTANCE_TYPE="${INSTANCE_TYPE:-t3.micro}"
DB_PASSWORD="${DB_PASSWORD:-hunter2pass}"

echo "🚀 Deploying test environment for RDS MySQL/MariaDB IAM authentication"

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
    --parameters \
        ParameterKey=KeyName,ParameterValue="$KEY_NAME" \
        ParameterKey=InstanceType,ParameterValue="$INSTANCE_TYPE" \
        ParameterKey=DBMasterPassword,ParameterValue="$DB_PASSWORD" \
    --capabilities CAPABILITY_NAMED_IAM

echo "⏳ Waiting for stack creation to complete (this may take 15-20 minutes)..."
aws cloudformation wait stack-create-complete \
    --stack-name "$STACK_NAME"

echo "✅ Stack created successfully!"

# Get outputs
AURORA_SERVERLESS_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`AuroraServerlessEndpoint`].OutputValue' \
    --output text)

AURORA_STANDARD_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`AuroraStandardEndpoint`].OutputValue' \
    --output text)

RDS_MARIADB_ENDPOINT=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`RDSMariaDBEndpoint`].OutputValue' \
    --output text)

EC2_IP=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`EC2InstancePublicIP`].OutputValue' \
    --output text)

EC2_ID=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`EC2InstanceId`].OutputValue' \
    --output text)

DB_USERNAME=$(aws cloudformation describe-stacks \
    --stack-name "$STACK_NAME" \
    --query 'Stacks[0].Outputs[?OutputKey==`DBMasterUsername`].OutputValue' \
    --output text)

echo "📋 Stack Outputs:"
echo "  Aurora Serverless Endpoint: $AURORA_SERVERLESS_ENDPOINT"
echo "  Aurora Standard Endpoint: $AURORA_STANDARD_ENDPOINT"
echo "  RDS MariaDB Endpoint: $RDS_MARIADB_ENDPOINT"
echo "  EC2 Public IP: $EC2_IP"
echo "  EC2 Instance ID: $EC2_ID"
echo "  DB Master Username: $DB_USERNAME"
echo "  SSH Key: ${KEY_NAME}.pem"

# Wait for instance to be ready
echo "⏳ Waiting for EC2 instance to be ready..."
aws ec2 wait instance-status-ok \
    --instance-ids "$EC2_ID"

# Save connection info
cat > test-env-info.txt << EOF
AURORA_SERVERLESS_ENDPOINT=$AURORA_SERVERLESS_ENDPOINT
AURORA_STANDARD_ENDPOINT=$AURORA_STANDARD_ENDPOINT
RDS_MARIADB_ENDPOINT=$RDS_MARIADB_ENDPOINT
EC2_IP=$EC2_IP
EC2_ID=$EC2_ID
DB_USERNAME=$DB_USERNAME
DB_PASSWORD=$DB_PASSWORD
SSH_KEY=${KEY_NAME}.pem
EOF

echo "✅ Test environment deployed successfully!"
echo ""
echo "🔐 Creating IAM database users..."

# Function to create IAM user in database
create_iam_user() {
    local endpoint=$1
    local engine=$2
    local user_name="fleet_iam_user"
    
    echo "  Creating IAM user on $endpoint..."
    
    # Use mysql client from EC2 instance
    ssh -o StrictHostKeyChecking=no -i "${KEY_NAME}.pem" ec2-user@$EC2_IP << EOF
        mysql -h"$endpoint" -u"$DB_USERNAME" -p"$DB_PASSWORD" -e "
            CREATE USER IF NOT EXISTS '$user_name'@'%' IDENTIFIED WITH AWSAuthenticationPlugin AS 'RDS';
            GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, INDEX, ALTER ON fleet.* TO '$user_name'@'%';
            FLUSH PRIVILEGES;
        " 2>/dev/null || echo "    Warning: Could not create IAM user on $endpoint"
EOF
}

# Wait a bit for databases to be fully ready
echo "  Waiting for databases to be ready..."
sleep 30

# Create users on all instances
create_iam_user "$AURORA_SERVERLESS_ENDPOINT" "mysql"
create_iam_user "$AURORA_STANDARD_ENDPOINT" "mysql"
create_iam_user "$RDS_MARIADB_ENDPOINT" "mariadb"

echo "✅ IAM users created!"
echo ""
echo "📝 Next steps:"
echo "1. Cross-compile the test binary:"
echo "   GOOS=linux GOARCH=amd64 go build -o iam_auth ./iam_auth.go"
echo ""
echo "2. Copy the binary to EC2:"
echo "   scp -i ${KEY_NAME}.pem iam_auth ec2-user@$EC2_IP:~/"
echo ""
echo "3. SSH to the instance:"
echo "   ssh -i ${KEY_NAME}.pem ec2-user@$EC2_IP"
echo ""
echo "4. Run the tests:"
echo "   # Test Aurora Serverless v2"
echo "   ./iam_auth -endpoint=$AURORA_SERVERLESS_ENDPOINT -user=fleet_iam_user"
echo ""
echo "   # Test Aurora Standard"
echo "   ./iam_auth -endpoint=$AURORA_STANDARD_ENDPOINT -user=fleet_iam_user"
echo ""
echo "   # Test RDS MariaDB"
echo "   ./iam_auth -endpoint=$RDS_MARIADB_ENDPOINT -user=fleet_iam_user"
echo ""
echo "To clean up when done:"
echo "   ./cleanup-test-env.sh"
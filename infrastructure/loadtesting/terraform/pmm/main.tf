terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
  }

  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/pmm/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = {
      environment = "loadtest-pmm"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform/pmm"
      workspace   = terraform.workspace
    }
  }
}

# Read shared VPC from remote state
data "terraform_remote_state" "shared" {
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/shared/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
  workspace = terraform.workspace
}

# Read infra state for RDS connection info
data "terraform_remote_state" "infra" {
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
  workspace = terraform.workspace
}

locals {
  customer = "fleet-${terraform.workspace}"
  # Private Subnets from VPN VPC
  vpn_cidr_blocks = [
    "10.255.1.0/24",
    "10.255.2.0/24",
    "10.255.3.0/24",
  ]
}

# DNS zone for vanity URL
data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}

# Latest Amazon Linux 2023 AMI
data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023*-x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

# RDS master password from Secrets Manager
data "aws_secretsmanager_secret" "rds_password" {
  name = "${local.customer}-database-password"
}

data "aws_secretsmanager_secret_version" "rds_password" {
  secret_id = data.aws_secretsmanager_secret.rds_password.id
}

# IAM role for the PMM EC2 instance
resource "aws_iam_role" "pmm" {
  name = "${local.customer}-pmm"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
  })
}

# Allow SSM Session Manager access (for debugging, no SSH needed)
resource "aws_iam_role_policy_attachment" "pmm_ssm" {
  role       = aws_iam_role.pmm.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "pmm" {
  name = "${local.customer}-pmm"
  role = aws_iam_role.pmm.name
}

# Security group: VPC + VPN access only on port 443
resource "aws_security_group" "pmm" {
  name_prefix = "${local.customer}-pmm-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "PMM server - HTTPS access from VPC and VPN only"

  # HTTPS UI + API from VPC
  ingress {
    description = "HTTPS from VPC"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [data.terraform_remote_state.shared.outputs.vpc.vpc_cidr_block]
  }

  # HTTPS from VPN
  ingress {
    description = "HTTPS from VPN"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = local.vpn_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

# PMM server EC2 instance
resource "aws_instance" "pmm" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = "t3.large"
  subnet_id              = data.terraform_remote_state.shared.outputs.vpc.private_subnets[0]
  vpc_security_group_ids = [aws_security_group.pmm.id]
  iam_instance_profile   = aws_iam_instance_profile.pmm.name

  root_block_device {
    volume_size = 50
    volume_type = "gp3"
    encrypted   = true
  }

  user_data = templatefile("${path.module}/user-data.sh", {
    rds_endpoint = data.terraform_remote_state.infra.outputs.rds_cluster_endpoint
    rds_username = data.terraform_remote_state.infra.outputs.rds_cluster_master_username
    rds_password = data.aws_secretsmanager_secret_version.rds_password.secret_string
  })

  tags = {
    Name = "${local.customer}-pmm"
  }
}

# Vanity DNS record (resolves to private IP, accessible via VPN)
resource "aws_route53_record" "pmm" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "pmm.${terraform.workspace}.loadtest.fleetdm.com"
  type    = "A"
  ttl     = 300
  records = [aws_instance.pmm.private_ip]
}

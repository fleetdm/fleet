terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
  }

  backend "s3" {
    bucket         = "fleet-terraform-state20220408141538466600000002"
    key            = "loadtesting/shared/eks-vpc/terraform.tfstate"
    region         = "us-east-2"
    encrypt        = true
    kms_key_id     = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = terraform.workspace
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform/eks-vpc"
      state       = "s3://fleet-terraform-state20220408141538466600000002/loadtesting/${terraform.workspace}/loadtesting/eks-vpc/terraform.tfstate"
    }
  }
}

# Shared VPC for EKS workloads with proper Kubernetes tags
# This VPC is shared across all workspaces (like fleet-vpc)
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "eks-vpc"
  cidr = "10.20.0.0/16"

  azs             = ["us-east-2a", "us-east-2b"]
  private_subnets = ["10.20.1.0/24", "10.20.2.0/24"]
  public_subnets  = ["10.20.101.0/24", "10.20.102.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  # Tags required for EKS - role tags are required on subnets
  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }

  # Note: Kubernetes cluster-specific tags are added by the signoz module
  # when creating each EKS cluster, not at the VPC level
  tags = {
    "shared" = "true"
  }
}

variable "region" {
  default = "us-east-2"
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      environment = "fleet-demo-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/tools/demo-environment"
      state       = "s3://fleet-loadtesting-tfstate/demo-environment"
    }
  }
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.74.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
    }
    git = {
      source  = "paultyng/git"
      version = "~> 0.1.0"
    }
  }
  backend "s3" {
    bucket         = "fleet-loadtesting-tfstate"
    key            = "demo-environment"
    region         = "us-east-2"
    dynamodb_table = "fleet-loadtesting-tfstate"
  }
}

data "aws_caller_identity" "current" {}

provider "git" {}

data "git_repository" "tf" {
  path = "${path.module}/../../"
}

locals {
  prefix = "fleet-demo"
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.12.0"

  name = local.prefix
  cidr = "10.11.0.0/16"

  azs                 = ["us-east-2a", "us-east-2b", "us-east-2c"]
  private_subnets     = ["10.11.1.0/24", "10.11.2.0/24", "10.11.3.0/24"]
  public_subnets      = ["10.11.11.0/24", "10.11.12.0/24", "10.11.13.0/24"]
  database_subnets    = ["10.11.21.0/24", "10.11.22.0/24", "10.11.23.0/24"]
  elasticache_subnets = ["10.11.31.0/24", "10.11.32.0/24", "10.11.33.0/24"]

  create_database_subnet_group       = false
  create_database_subnet_route_table = true

  create_elasticache_subnet_group       = true
  create_elasticache_subnet_route_table = true

  enable_vpn_gateway     = false
  one_nat_gateway_per_az = false

  single_nat_gateway = true
  enable_nat_gateway = true
}

module "shared-infrastructure" {
  source              = "./SharedInfrastructure"
  prefix              = local.prefix
  vpc_id              = module.vpc.vpc_id
  database_subnets    = module.vpc.database_subnets
  allowed_cidr_blocks = module.vpc.private_subnets_cidr_blocks
  private_subnets     = module.vpc.private_subnets
}

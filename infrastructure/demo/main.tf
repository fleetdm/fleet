terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.10.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
    }
    git = {
      source  = "paultyng/git"
      version = "~> 0.1.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1.2"
    }
  }
  backend "s3" {
    bucket         = "fleet-loadtesting-tfstate"
    key            = "demo-environment"
    region         = "us-east-2"
    dynamodb_table = "fleet-loadtesting-tfstate"
  }
}


provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "fleet-demo-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/demo"
      state       = "s3://fleet-loadtesting-tfstate/demo-environment"
    }
  }
}
provider "aws" {
  alias  = "replica"
  region = "us-west-1"
  default_tags {
    tags = {
      environment = "fleet-demo-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/demo"
      state       = "s3://fleet-loadtesting-tfstate/demo-environment"
    }
  }
}


provider "random" {}

data "aws_ecr_authorization_token" "token" {}
provider "docker" {
  # Configuration options
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.us-east-2.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

provider "git" {}

data "aws_caller_identity" "current" {}

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
  source                  = "./SharedInfrastructure"
  prefix                  = local.prefix
  vpc                     = module.vpc
  allowed_security_groups = [module.pre-provisioner.lambda_security_group.id]
  eks_allowed_roles       = [module.pre-provisioner.lambda_role]
}

module "pre-provisioner" {
  source         = "./PreProvisioner"
  prefix         = local.prefix
  vpc            = module.vpc
  dynamodb_table = aws_dynamodb_table.lifecycle-table
  remote_state   = module.remote_state
  mysql_secret   = module.shared-infrastructure.mysql_secret
  eks_cluster    = module.shared-infrastructure.eks_cluster
  redis_cluster  = module.shared-infrastructure.redis_cluster
}

module "jit-provisioner" {
  source         = "./JITProvisioner"
  prefix         = local.prefix
  vpc            = module.vpc
  dynamodb_table = aws_dynamodb_table.lifecycle-table
  remote_state   = module.remote_state
  mysql_secret   = module.shared-infrastructure.mysql_secret
  eks_cluster    = module.shared-infrastructure.eks_cluster
  redis_cluster  = module.shared-infrastructure.redis_cluster
}

resource "aws_dynamodb_table" "lifecycle-table" {
  name         = "${local.prefix}-lifecycle"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "ID"
  range_key    = "State"

  attribute {
    name = "ID"
    type = "S"
  }

  attribute {
    name = "State"
    type = "S"
  }

  attribute {
    name = "redis_db"
    type = "N"
  }

  global_secondary_index {
    name            = "RedisDatabases"
    hash_key        = "redis_db"
    projection_type = "KEYS_ONLY"
  }
}

module "remote_state" {
  source = "nozaq/remote-state-s3-backend/aws"
  tags   = {}

  providers = {
    aws         = aws
    aws.replica = aws.replica
  }
}

resource "aws_ecs_cluster" "main" {
  name = local.prefix

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

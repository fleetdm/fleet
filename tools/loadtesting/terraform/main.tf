variable "region" {
  default = "us-east-2"
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      environment = "loadtest"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/tools/terraform"
      state       = "local"
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
    key            = "loadtesting"
    region         = "us-east-2"
    dynamodb_table = "fleet-loadtesting-tfstate"
  }
}

data "aws_caller_identity" "current" {}

provider "docker" {
  # Configuration options
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.us-east-2.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

provider "git" {}

data "git_repository" "tf" {
  path = "${path.module}/../../../"
}

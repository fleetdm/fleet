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
  // these values are hard-coded to prevent chicken before the egg situations
  #  backend "s3" {
  #    bucket = "fleet-terraform-remote-state"
  #    region = "us-east-2"
  #    key = "fleet/"
  #    dynamodb_table = "fleet-terraform-state-lock"
  #  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.74.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
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

#resource "aws_s3_bucket" "remote_state" {
#  bucket = "${var.prefix}-terraform-remote-state"
#  acl    = "private"
#  versioning {
#    enabled = true
#  }
#  lifecycle {
#    prevent_destroy = true
#  }
#  tags = {
#    Name = "S3 Remote Terraform State Store"
#  }
#}
#
#resource "aws_s3_bucket_public_access_block" "fleet_terraform_state" {
#  bucket              = aws_s3_bucket.remote_state.id
#  block_public_acls   = true
#  block_public_policy = true
#}
#
#resource "aws_dynamodb_table" "fleet_terraform_state_lock" {
#  name         = "fleet-terraform-state-lock"
#  hash_key     = "LockID"
#  billing_mode = "PAY_PER_REQUEST"
#
#  attribute {
#    name = "LockID"
#    type = "S"
#  }
#
#  tags = {
#    Name = "DynamoDB Terraform State Lock Table"
#  }
#}

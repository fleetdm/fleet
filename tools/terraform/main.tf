variable "region" {
  default = "us-east-2"
}

provider "aws" {
  region = var.region
}

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "3.54.0"
    }
  }
}

data aws_caller_identity "current" {}


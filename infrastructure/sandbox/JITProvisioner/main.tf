terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 2.16.0"
    }
    git = {
      source  = "paultyng/git"
      version = "~> 0.1.0"
    }
  }
}

data "aws_region" "current" {}

locals {
  name      = "jit"
  full_name = "${var.prefix}-${local.name}"
}

resource "aws_cloudwatch_log_group" "main" {
  name              = local.full_name
  kms_key_id        = var.kms_key.arn
  retention_in_days = 30
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "main" {
  name                 = var.prefix
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }
}

data "git_repository" "main" {
  path = "${path.module}/../../../"
}

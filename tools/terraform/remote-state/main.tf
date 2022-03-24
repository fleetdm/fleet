variable "prefix" {
  default = "fleet"
}

variable "region" {
  default = "us-east-2"
}

provider "aws" {
  region = var.region
}

resource "aws_s3_bucket" "remote_state" {
  bucket = "${var.prefix}-terraform-remote-state"
  acl    = "private"
  versioning {
    enabled = true
  }
  lifecycle {
    prevent_destroy = true
  }
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "aws:kms"
      }
    }
  }
  tags = {
    Name = "S3 Remote Terraform State Store"
  }
}

resource "aws_s3_bucket_public_access_block" "fleet_terraform_state" {
  bucket              = aws_s3_bucket.remote_state.id
  block_public_acls   = true
  block_public_policy = true
  ignore_public_acls = true
  restrict_public_buckets = true 
}


resource "aws_dynamodb_table" "fleet_terraform_state_lock" {
  name         = "${var.prefix}-terraform-state-lock"
  hash_key     = "LockID"
  billing_mode = "PAY_PER_REQUEST"

  attribute {
    name = "LockID"
    type = "S"
  }

  tags = {
    Name = "DynamoDB Terraform State Lock Table"
  }

  server_side_encryption {
    enabled = true // enabled server side encryption
    }
  
  point_in_time_recovery {
        enabled = true
    }
}
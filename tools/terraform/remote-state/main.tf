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
  tags = {
    Name = "S3 Remote Terraform State Store"
  }
}

resource "aws_s3_bucket_public_access_block" "fleet_terraform_state" {
  bucket              = aws_s3_bucket.remote_state.id
  block_public_acls   = true
  block_public_policy = true
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
}
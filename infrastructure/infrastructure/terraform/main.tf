provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "loadtest"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/terraform"
      state       = "local"
    }
  }
}

provider "aws" {
  alias  = "replica"
  region = "us-west-1"
  default_tags {
    tags = {
      environment = "loadtest"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/terraform"
      state       = "local"
    }
  }
}

data "aws_caller_identity" "current" {}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.9.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "infrastructure/state/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "infrastructure"                         # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
  }
}

locals {
  accounts = {
    frontend-loadtesting     = "851787985745"
    dogfood                  = "160035666661"
    loadtesting              = "917007347864"
    root                     = "831217569274"
    fleet-cloud              = "611884880216"
    fleet-try                = "564445215450"
  }
}

module "remote-state-s3-backend" {
  source                                 = "nozaq/remote-state-s3-backend/aws"
  version                                = "1.1.2"
  dynamodb_enable_server_side_encryption = true
  state_bucket_prefix                    = "fleet-terraform-state"
  tags                                   = {}
  providers = {
    aws         = aws
    aws.replica = aws.replica
  }
}

data "aws_iam_policy_document" "assume-role-policy" {
  for_each = local.accounts
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "AWS"
      identifiers = [each.value]
    }
  }
}

resource "aws_iam_role" "main" {
  for_each           = local.accounts
  name               = "terraform-${each.key}"
  assume_role_policy = data.aws_iam_policy_document.assume-role-policy[each.key].json
}

resource "aws_iam_role_policy_attachment" "main" {
  for_each   = local.accounts
  role       = aws_iam_role.main[each.key].name
  policy_arn = aws_iam_policy.main[each.key].arn
}

resource "aws_iam_policy" "main" {
  for_each = local.accounts
  name     = "terraform-${each.key}"
  policy   = data.aws_iam_policy_document.main[each.key].json
}

data "aws_iam_policy_document" "main" {
  for_each = local.accounts
  statement {
    actions = [
      "s3:ListBucket",
      "s3:GetBucketVersioning",
    ]
    resources = [module.remote-state-s3-backend.state_bucket.arn]
  }
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject",
    ]
    resources = ["${module.remote-state-s3-backend.state_bucket.arn}/${each.key}/*"]
  }
  statement {
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:DeleteItem",
      "dynamodb:DescribeTable",
    ]
    resources = [module.remote-state-s3-backend.dynamodb_table.arn]
  }
  statement {
    actions = [
      "kms:ListKeys"
    ]
    resources = ["*"]
  }
  statement {
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:GenerateDataKey"
    ]
    resources = [module.remote-state-s3-backend.kms_key.arn]
  }
}

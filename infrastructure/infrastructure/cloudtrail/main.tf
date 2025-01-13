terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.62.1"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "root/cloudtrail/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "root"                              # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    role_arn             = "arn:aws:iam::353365949058:role/terraform-root"
  }
}

provider "aws" {
  default_tags {
    tags = {
      environment = "cloudtrail"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/cloudtrail"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/cloudtrail/terraform.tfstate"
    }
  }
}

provider "aws" {
  region = "us-east-2"
  alias  = "security"
  assume_role {
    role_arn = "arn:aws:iam::353365949058:role/admin"
  }
  default_tags {
    tags = {
      environment = "cloudtrail"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/cloudtrail"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/cloudtrail/terraform.tfstate"
    }
  }
}

data "aws_organizations_organization" "main" {}
data "aws_caller_identity" "current" {}

module "cloudtrail" {
  providers = {
    aws = aws.security
  }
  source = "terraform-aws-modules/s3-bucket/aws"

  bucket                  = "fleet-cloudtrail-logs"
  acl                     = "private"
  block_public_policy     = true
  block_public_acls       = true
  restrict_public_buckets = true
  ignore_public_acls      = true

  attach_policy = true
  policy        = data.aws_iam_policy_document.cloudtrail.json

  versioning = {
    enabled = true
  }

}

data "aws_iam_policy_document" "cloudtrail" {
  statement {
    resources = [module.cloudtrail.s3_bucket_arn]
    actions   = ["s3:GetBucketAcl"]
    principals {
      type        = "Service"
      identifiers = ["cloudtrail.amazonaws.com"]
    }
    #condition {
    #  test     = "StringEquals"
    #  variable = "aws:SourceArn"

    #  values = formatlist("arn:aws:cloudtrail:*:%s:trail/cloudtrail", data.aws_organizations_organization.main.accounts.*.id)
    #}
  }
  statement {
    resources = ["${module.cloudtrail.s3_bucket_arn}/*"]
    actions   = ["s3:PutObject"]
    principals {
      type        = "Service"
      identifiers = ["cloudtrail.amazonaws.com"]
    }
    #condition {
    #  test     = "StringEquals"
    #  variable = "aws:SourceArn"

    #  values = formatlist("arn:aws:cloudtrail:*:%s:trail/cloudtrail", data.aws_organizations_organization.main.accounts.*.id)
    #}
  }
}

resource "aws_cloudtrail" "main" {
  name                       = "cloudtrail"
  s3_bucket_name             = module.cloudtrail.s3_bucket_id
  s3_key_prefix              = data.aws_caller_identity.current.account_id
  is_multi_region_trail      = true
  enable_log_file_validation = true
  is_organization_trail      = true
}

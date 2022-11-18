terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.10.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "root/guardduty/members/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "root"                                     # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    role_arn             = "arn:aws:iam::353365949058:role/terraform-root"
  }
}

provider "aws" {
  region = local.region
  alias  = "security"
  assume_role {
    role_arn = "arn:aws:iam::353365949058:role/admin"
  }
  default_tags {
    tags = {
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty/members"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/members/terraform.tfstate"
    }
  }
}

provider "aws" {
  region = local.region
  alias  = "member"
  assume_role {
    role_arn = "arn:aws:iam::${local.account_id}:role/admin"
  }
  default_tags {
    tags = {
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty/members"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/members/terraform.tfstate"
    }
  }
}

provider "aws" {
  region = local.region
  alias  = "root"
  default_tags {
    tags = {
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty/members"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/members/terraform.tfstate"
    }
  }
}

locals {
  account_id = split(":", terraform.workspace)[0]
  region     = split(":", terraform.workspace)[1]
  accounts   = { for i in data.aws_organizations_organization.main.non_master_accounts : i.id => i.email }
}

data "aws_organizations_organization" "main" {
  provider = aws.root
}

resource "aws_guardduty_member" "member" {
  provider                   = aws.security
  account_id                 = aws_guardduty_detector.member.account_id
  detector_id                = data.aws_guardduty_detector.security.id
  email                      = local.accounts[local.account_id]
  disable_email_notification = true
  invite                     = true
  lifecycle {
    ignore_changes = [email]
  }
}

resource "aws_guardduty_detector" "member" {
  provider = aws.member
}

data "aws_guardduty_detector" "security" {
  provider = aws.security
}

data "aws_caller_identity" "security" {}

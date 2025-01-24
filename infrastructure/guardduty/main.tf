terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.62.3"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "root/guardduty/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "root"                             # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    role_arn             = "arn:aws:iam::353365949058:role/terraform-root"
  }
}

data "terraform_remote_state" "findings" {
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "root/guardduty/findings/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "root"                                      # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    role_arn             = "arn:aws:iam::353365949058:role/terraform-root"
  }
}

provider "aws" {
  region = terraform.workspace
  default_tags {
    tags = {
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/terraform.tfstate"
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
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/terraform.tfstate"
    }
  }
}

provider "aws" {
  region = terraform.workspace
  alias  = "security-region"
  assume_role {
    role_arn = "arn:aws:iam::353365949058:role/admin"
  }
  default_tags {
    tags = {
      environment = "guardduty-${terraform.workspace}"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/guardduty"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/guardduty/terraform.tfstate"
    }
  }
}

resource "aws_guardduty_organization_admin_account" "main" {
  admin_account_id = "353365949058"
}

data "aws_guardduty_detector" "main" {
  provider = aws.security-region
}

data "aws_s3_bucket" "findings" {
  provider = aws.security
  bucket   = "fleet-guardduty-findings"
}

resource "aws_guardduty_publishing_destination" "main" {
  provider        = aws.security-region
  detector_id     = data.aws_guardduty_detector.main.id
  destination_arn = data.aws_s3_bucket.findings.arn
  kms_key_arn     = data.terraform_remote_state.findings.outputs.kms_key.arn
}

resource "aws_guardduty_detector" "root" {}

data "aws_organizations_organization" "main" {}

resource "aws_guardduty_member" "root" {
  provider                   = aws.security-region
  account_id                 = aws_guardduty_detector.root.account_id
  detector_id                = data.aws_guardduty_detector.main.id
  email                      = data.aws_organizations_organization.main.master_account_email
  disable_email_notification = true
  invite                     = true
}

resource "aws_guardduty_organization_configuration" "main" {
  provider    = aws.security-region
  auto_enable = true
  detector_id = data.aws_guardduty_detector.main.id
}

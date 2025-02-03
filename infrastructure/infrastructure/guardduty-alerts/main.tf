provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "guardduty-alerts"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/guardduty-alerts"
      state       = "local"
    }
  }
}

data "aws_caller_identity" "current" {}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.62.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "infrastructure/guardduty-alerts/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "infrastructure"                                    # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
  }
}

variable "slack_webhook" {
  type = string
}

module "guardduty-to-sns" {
  source  = "rhythmictech/guardduty-to-sns/aws"
  version = "1.0.0-rc1"

  notification_arn = module.notify_slack.slack_topic_arn
}

module "notify_slack" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "6.0.0"

  sns_topic_name       = "guardduty-${terraform.workspace}"
  lambda_function_name = "guardduty-${terraform.workspace}"

  slack_webhook_url = var.slack_webhook
  slack_channel     = "#g-infra"
  slack_username    = "monitoring"
}

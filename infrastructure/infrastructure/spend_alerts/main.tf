terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.62.3"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "root/spend-alerts/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "root"                                # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    role_arn             = "arn:aws:iam::353365949058:role/terraform-root"
  }
}

provider "aws" {
  region = "us-east-1"
  default_tags {
    tags = {
      environment = "spend-alerts"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/spend_alerts"
      state       = "s3://fleet-terraform-state20220408141538466600000002/root/spend-alerts/terraform.tfstate"
      VantaOwner  = "robert@fleetdm.com"
    }
  }
}

variable "slack_webhook" {
  type = string
}

locals {
  prefix = "aws-spend-alerts"
}

module "notify_slack" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "6.0.0"

  sns_topic_name       = local.prefix
  lambda_function_name = local.prefix

  slack_webhook_url = var.slack_webhook
  slack_channel     = "#g-infra"
  slack_username    = "monitoring"
}

output "slack_topic_arn" {
  value = module.notify_slack.slack_topic_arn
}

resource "aws_cloudwatch_metric_alarm" "total_charge" {
  alarm_name                = "total_charge"
  alarm_description         = "total estimated charge"
  comparison_operator       = "GreaterThanUpperThreshold"
  evaluation_periods        = "1"
  threshold_metric_id       = "ad1"
  alarm_actions             = [module.notify_slack.slack_topic_arn]
  ok_actions                = [module.notify_slack.slack_topic_arn]
  insufficient_data_actions = []

  metric_query {
    id          = "m1"
    period      = 0
    return_data = true

    metric {
      dimensions = {
        "Currency" = "USD"
      }
      metric_name = "EstimatedCharges"
      namespace   = "AWS/Billing"
      period      = 86400
      stat        = "Maximum"
    }
  }

  metric_query {
    expression  = "ANOMALY_DETECTION_BAND(m1, 2)"
    id          = "ad1"
    label       = "EstimatedCharges (expected)"
    period      = 0
    return_data = true
  }
}

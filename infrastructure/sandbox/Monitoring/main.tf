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
  full_name = "${var.prefix}-monitoring"
}

module "notify_slack" {
  source  = "terraform-aws-modules/notify-slack/aws"
  version = "5.5.0"

  sns_topic_name = var.prefix

  slack_webhook_url = var.slack_webhook
  slack_channel     = "#help-p1"
  slack_username    = "monitoring"
}

data "aws_iam_policy_document" "lifecycle-lambda-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "lifecycle-lambda-lambda" {
  role       = aws_iam_role.lifecycle-lambda.id
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy_attachment" "lifecycle-lambda" {
  role       = aws_iam_role.lifecycle-lambda.id
  policy_arn = aws_iam_policy.lifecycle-lambda.arn
}

resource "aws_iam_policy" "lifecycle-lambda" {
  name   = "${local.full_name}-lifecycle-lambda"
  policy = data.aws_iam_policy_document.lifecycle-lambda.json
}

data "aws_iam_policy_document" "lifecycle-lambda" {
  statement {
    actions = [
      "dynamodb:List*",
      "dynamodb:DescribeReservedCapacity*",
      "dynamodb:DescribeLimits",
      "dynamodb:DescribeTimeToLive"
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "dynamodb:BatchGet*",
      "dynamodb:DescribeStream",
      "dynamodb:DescribeTable",
      "dynamodb:Get*",
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:BatchWrite*",
      "dynamodb:CreateTable",
      "dynamodb:Delete*",
      "dynamodb:Update*",
      "dynamodb:PutItem"
    ]
    resources = [var.dynamodb_table.arn]
  }

  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [aws_kms_key.ecr.arn, var.kms_key.arn]
  }

  statement {
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "lifecycle-lambda" {
  name = local.full_name

  assume_role_policy = data.aws_iam_policy_document.lifecycle-lambda-assume-role.json
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "main" {
  name                 = local.full_name
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }
}

resource "random_uuid" "lifecycle-lambda" {
  keepers = {
    lambda = data.archive_file.lifecycle-lambda.output_sha
  }
}

data "archive_file" "lifecycle-lambda" {
  type        = "zip"
  output_path = "${path.module}/.lambda.zip"
  source_dir  = "${path.module}/lambda"
}

data "git_repository" "main" {
  path = "${path.module}/../../../"
}

resource "docker_registry_image" "lifecycle-lambda" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.lifecycle-lambda.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/lambda/"
    pull_parent = true
    platform    = "linux/amd64"
  }
}

resource "aws_cloudwatch_event_rule" "lifecycle" {
  name_prefix         = local.full_name
  schedule_expression = "rate(5 minutes)"
  is_enabled          = true
}

resource "aws_cloudwatch_event_target" "lifecycle" {
  rule = aws_cloudwatch_event_rule.lifecycle.name
  arn  = aws_lambda_function.lifecycle.arn
}

resource "aws_lambda_function" "lifecycle" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  image_uri                      = docker_registry_image.lifecycle-lambda.name
  package_type                   = "Image"
  function_name                  = "${local.full_name}-lifecycle-lambda"
  kms_key_arn                    = var.kms_key.arn
  role                           = aws_iam_role.lifecycle-lambda.arn
  reserved_concurrent_executions = -1
  timeout                        = 10
  memory_size                    = 512
  tracing_config {
    mode = "Active"
  }
  environment {
    variables = {
      DYNAMODB_LIFECYCLE_TABLE = var.dynamodb_table.id
    }
  }
}

resource "aws_lambda_permission" "lifecycle" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.lifecycle.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.lifecycle.arn
}

resource "aws_cloudwatch_metric_alarm" "totalInstances" {
  alarm_name          = "${var.prefix}-lifecycle-totalCount"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "instances"
  namespace           = "Fleet/sandbox"
  period              = "900"
  statistic           = "Average"
  threshold           = "90"
  alarm_actions       = [module.notify_slack.slack_topic_arn]
  ok_actions          = [module.notify_slack.slack_topic_arn]
  treat_missing_data  = "breaching"
  datapoints_to_alarm = 1
  dimensions = {
    Type = "totalCount"
  }
}

resource "aws_cloudwatch_metric_alarm" "unclaimed" {
  alarm_name          = "${var.prefix}-lifecycle-unclaimed"
  comparison_operator = "LessThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "instances"
  namespace           = "Fleet/sandbox"
  period              = "900"
  statistic           = "Average"
  threshold           = "10"
  alarm_actions       = [module.notify_slack.slack_topic_arn]
  ok_actions          = [module.notify_slack.slack_topic_arn]
  treat_missing_data  = "breaching"
  datapoints_to_alarm = 1
  dimensions = {
    Type = "unclaimedCount"
  }
}

resource "aws_cloudwatch_metric_alarm" "lb" {
  for_each            = toset(["HTTPCode_ELB_5XX_Count", "HTTPCode_Target_5XX_Count"])
  alarm_name          = "${var.prefix}-lb-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = each.key
  namespace           = "AWS/ApplicationELB"
  period              = "120"
  statistic           = "Sum"
  threshold           = "0"
  alarm_actions       = [module.notify_slack.slack_topic_arn]
  ok_actions          = [module.notify_slack.slack_topic_arn]
  treat_missing_data  = "notBreaching"
  dimensions = {
    LoadBalancer = var.lb.arn_suffix
  }
}

resource "aws_cloudwatch_metric_alarm" "jitprovisioner" {
  for_each            = toset(["Errors"])
  alarm_name          = "${var.prefix}-jitprovisioner-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = each.key
  namespace           = "AWS/Lambda"
  period              = "120"
  statistic           = "Sum"
  threshold           = "0"
  alarm_actions       = [module.notify_slack.slack_topic_arn]
  ok_actions          = [module.notify_slack.slack_topic_arn]
  treat_missing_data  = "notBreaching"
  dimensions = {
    FunctionName = var.jitprovisioner.id
  }
}

resource "aws_cloudwatch_metric_alarm" "deprovisioner" {
  for_each            = toset(["ExecutionsFailed"])
  alarm_name          = "${var.prefix}-deprovisioner-${each.key}"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = each.key
  namespace           = "AWS/States"
  period              = "120"
  statistic           = "Sum"
  threshold           = "0"
  alarm_actions       = [module.notify_slack.slack_topic_arn]
  ok_actions          = [module.notify_slack.slack_topic_arn]
  treat_missing_data  = "notBreaching"
  dimensions = {
    StateMachineArn = var.deprovisioner.arn
  }
}

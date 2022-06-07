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
  name      = "jitprovisioner"
  full_name = "${var.prefix}-${local.name}"
}

resource "aws_cloudwatch_log_group" "main" {
  name = local.full_name
}

data "aws_iam_policy_document" "sfn-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "sfn" {
  role       = aws_iam_role.sfn.id
  policy_arn = aws_iam_policy.sfn.arn
}

resource "aws_iam_policy" "sfn" {
  name   = "${var.prefix}-jitprovisioner-sfn"
  policy = data.aws_iam_policy_document.sfn.json
}

data "aws_iam_policy_document" "sfn" {
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
    resources = [aws_kms_key.ecr.arn]
  }

  statement {
    actions   = ["*"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "sfn" {
  name = "${var.prefix}-jitprovisioner-sfn"

  assume_role_policy = data.aws_iam_policy_document.sfn-assume-role.json
}

resource "aws_iam_role_policy_attachment" "deprovisioner" {
  role       = aws_iam_role.deprovisioner.id
  policy_arn = aws_iam_policy.deprovisioner.arn
}

resource "aws_iam_policy" "deprovisioner" {
  name   = "${var.prefix}-deprovisioner"
  policy = data.aws_iam_policy_document.sfn.json
}

data "aws_iam_policy_document" "deprovisioner" {
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
    resources = [aws_kms_key.ecr.arn]
  }

  statement {
    actions   = ["*"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "deprovisioner" {
  name = "${var.prefix}-deprovisioner"

  assume_role_policy = data.aws_iam_policy_document.sfn-assume-role.json
}

resource "aws_ecs_task_definition" "deprovisioner" {
  family                   = "${var.prefix}-deprovisioner"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.deprovisioner.arn
  task_role_arn            = aws_iam_role.deprovisioner.arn
  cpu                      = 1024
  memory                   = 4096
  container_definitions = jsonencode(
    [
      {
        name        = "${var.prefix}-deprovisioner"
        image       = docker_registry_image.deprovisioner.name
        mountPoints = []
        volumesFrom = []
        essential   = true
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.main.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "${var.prefix}-deprovisioner"
          }
        },
        environment = concat([
          {
            name  = "DYNAMODB_LIFECYCLE_TABLE"
            value = var.dynamodb_table.id
          },
        ])
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_sfn_state_machine" "main" {
  name     = var.prefix
  role_arn = aws_iam_role.sfn.arn

  definition = <<EOF
{
  "Comment": "Controls the lifecycle of a Fleet demo environment",
  "StartAt": "Wait",
  "States": {
    "Wait": {
      "Type": "Wait",
      "Seconds": 5,
      "Next": "Deprovisioner"
    },
    "Deprovisioner": {
      "Type": "Task",
      "Resource": "arn:aws:states:::ecs:runTask.sync",
      "Parameters": {
        "LaunchType": "FARGATE",
        "Cluster": "arn:aws:ecs:REGION:ACCOUNT_ID:cluster/MyECSCluster",
        "TaskDefinition": "${aws_ecs_task_definition.deprovisioner.arn}"
      },
      "End": true
    }
  }
}
EOF
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
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

resource "random_uuid" "deprovisioner" {
  keepers = {
    lambda = data.archive_file.deprovisioner.output_sha
  }
}

data "archive_file" "deprovisioner" {
  type        = "zip"
  output_path = "${path.module}/.deprovisioner.zip"
  source_dir  = "${path.module}/deprovisioner"
}


data "git_repository" "main" {
  path = "${path.module}/../../../"
}

resource "docker_registry_image" "deprovisioner" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.deprovisioner.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/lambda/"
    pull_parent = true
  }
}

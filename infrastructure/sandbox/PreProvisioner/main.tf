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

data "aws_caller_identity" "current" {}

locals {
  name      = "preprovisioner"
  full_name = "${var.prefix}-${local.name}"
}

resource "aws_cloudwatch_log_group" "main" {
  name              = local.full_name
  kms_key_id        = var.kms_key.arn
  retention_in_days = 30
}

data "aws_iam_policy_document" "events-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "events" {
  role       = aws_iam_role.events.id
  policy_arn = aws_iam_policy.events.arn
}

resource "aws_iam_policy" "events" {
  name   = "${local.full_name}-events"
  policy = data.aws_iam_policy_document.events.json
}

data "aws_iam_policy_document" "events" {
  statement {
    actions   = ["ecs:RunTask"]
    resources = [replace(aws_ecs_task_definition.main.arn, "/:\\d+$/", ":*"), replace(aws_ecs_task_definition.main.arn, "/:\\d+$/", "")]
    condition {
      test     = "ArnLike"
      variable = "ecs:cluster"
      values   = [var.ecs_cluster.arn]
    }
  }
  statement {
    actions   = ["iam:PassRole"]
    resources = ["*"]
    condition {
      test     = "StringLike"
      variable = "iam:PassedToService"
      values   = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "events" {
  name = "${local.full_name}-events"
  path = "/service-role/"

  assume_role_policy = data.aws_iam_policy_document.events-assume-role.json
}

data "aws_iam_policy_document" "lambda-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}


resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.lambda.arn
}

resource "aws_iam_role_policy_attachment" "lambda-ecs" {
  role       = aws_iam_role.lambda.id
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSFargatePodExecutionRolePolicy"
}

resource "aws_iam_policy" "lambda" {
  name   = "${var.prefix}-lambda"
  policy = data.aws_iam_policy_document.lambda.json
}

data "aws_iam_policy_document" "lambda" {
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
    resources = [aws_kms_key.ecr.arn, var.kms_key.arn, var.installer_kms_key.arn]
  }

  statement {
    actions = [
      "s3:*Object",
      "s3:ListBucket",
    ]
    resources = [
      var.installer_bucket.arn,
      "${var.installer_bucket.arn}/*"
    ]
  }

  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.apple-signing-secrets.arn]
  }

  # TODO: limit this, this is for terraform
  statement {
    actions   = ["*"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "lambda" {
  name = local.full_name

  assume_role_policy = data.aws_iam_policy_document.lambda-assume-role.json
}

output "lambda_role" {
  value = aws_iam_role.lambda
}

resource "aws_security_group" "lambda" {
  name        = local.full_name
  description = "security group for ${local.full_name}"
  vpc_id      = var.vpc.vpc_id

  egress {
    description      = "egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

data "aws_eks_cluster" "cluster" {
  name = var.eks_cluster.eks_cluster_id
}

resource "aws_secretsmanager_secret" "apple-signing-secrets" {
  name                    = "${local.full_name}-apple-signing-secrets"
  kms_key_id              = var.kms_key.id
  recovery_window_in_days = 0
}

data "aws_secretsmanager_secret_version" "apple-signing-secrets" {
  secret_id = aws_secretsmanager_secret.apple-signing-secrets.id
}

resource "aws_ecs_task_definition" "main" {
  family                   = local.full_name
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.lambda.arn
  task_role_arn            = aws_iam_role.lambda.arn
  cpu                      = 1024
  memory                   = 4096
  container_definitions = jsonencode(
    [
      {
        name        = local.name
        image       = docker_registry_image.main.name
        mountPoints = []
        volumesFrom = []
        essential   = true
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.main.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = local.full_name
          }
        },
        environment = concat([
          {
            name  = "TF_VAR_mysql_secret"
            value = var.mysql_secret.id
          },
          {
            name  = "TF_VAR_mysql_cluster_name"
            value = var.eks_cluster.eks_cluster_id
          },
          {
            name  = "TF_VAR_eks_cluster"
            value = var.eks_cluster.eks_cluster_id
          },
          {
            name  = "DYNAMODB_LIFECYCLE_TABLE"
            value = var.dynamodb_table.id
          },
          {
            name  = "TF_VAR_lifecycle_table"
            value = var.dynamodb_table.id
          },
          {
            name  = "TF_VAR_base_domain"
            value = var.base_domain
          },
          {
            name  = "MAX_INSTANCES"
            value = "500"
          },
          {
            name  = "QUEUED_INSTANCES"
            value = "20"
          },
          {
            name  = "TF_VAR_redis_address"
            value = "${var.redis_cluster.primary_endpoint_address}:6379"
          },
          {
            name  = "FLEET_BASE_URL"
            value = var.base_domain
          },
          {
            name  = "INSTALLER_BUCKET"
            value = var.installer_bucket.id
          },
          {
            name  = "TF_VAR_installer_bucket"
            value = var.installer_bucket.id
          },
          {
            name  = "TF_VAR_installer_bucket_arn"
            value = var.installer_bucket.arn
          },
          {
            name  = "TF_VAR_oidc_provider_arn"
            value = var.oidc_provider_arn
          },
          {
            name  = "TF_VAR_oidc_provider"
            value = var.oidc_provider
          },
          {
            name  = "TF_VAR_kms_key_arn"
            value = var.installer_kms_key.arn
          },
          {
            name  = "TF_VAR_ecr_url"
            value = var.ecr.repository_url
          },
          {
            name  = "TF_VAR_license_key"
            value = var.license_key
          },
          {
            name  = "TF_VAR_apm_url"
            value = var.apm_url
          },
          {
            name  = "TF_VAR_apm_token"
            value = var.apm_token
          },
        ]),
        secrets = concat([
          {
            name      = "MACOS_DEV_ID_CERTIFICATE_CONTENT"
            valueFrom = "${aws_secretsmanager_secret.apple-signing-secrets.arn}:MACOS_DEV_ID_CERTIFICATE_CONTENT::"
          },
          {
            name      = "APP_STORE_CONNECT_API_KEY_ID"
            valueFrom = "${aws_secretsmanager_secret.apple-signing-secrets.arn}:APP_STORE_CONNECT_API_KEY_ID::"
          },
          {
            name      = "APP_STORE_CONNECT_API_KEY_ISSUER"
            valueFrom = "${aws_secretsmanager_secret.apple-signing-secrets.arn}:APP_STORE_CONNECT_API_KEY_ISSUER::"
          },
          {
            name      = "APP_STORE_CONNECT_API_KEY_CONTENT"
            valueFrom = "${aws_secretsmanager_secret.apple-signing-secrets.arn}:APP_STORE_CONNECT_API_KEY_CONTENT::"
          }
        ])
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "main" {
  name                 = "${var.prefix}-lambda"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.ecr.arn
  }
}

resource "random_uuid" "main" {
  keepers = {
    lambda = data.archive_file.main.output_sha
  }
}

resource "local_file" "backend-config" {
  content = templatefile("${path.module}/lambda/backend-template.conf",
    {
      remote_state = var.remote_state
  })
  filename = "${path.module}/lambda/deploy_terraform/backend.conf"
}

data "archive_file" "main" {
  type        = "zip"
  output_path = "${path.module}/.lambda.zip"
  source_dir  = "${path.module}/lambda"
}

data "git_repository" "main" {
  path = "${path.module}/../../../"
}

resource "docker_registry_image" "main" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.main.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/lambda/"
    pull_parent = true
    platform    = "linux/amd64"
  }

  depends_on = [
    local_file.backend-config
  ]
}

resource "aws_cloudwatch_event_rule" "main" {
  name_prefix         = var.prefix
  schedule_expression = "rate(1 hour)"
  is_enabled          = true
}

resource "aws_cloudwatch_event_target" "main" {
  rule     = aws_cloudwatch_event_rule.main.name
  arn      = var.ecs_cluster.arn
  role_arn = aws_iam_role.events.arn
  ecs_target {
    task_count          = 1
    task_definition_arn = aws_ecs_task_definition.main.arn
    launch_type         = "FARGATE"
    network_configuration {
      subnets          = var.vpc.private_subnets
      security_groups  = [aws_security_group.lambda.id]
      assign_public_ip = false
    }
  }
}

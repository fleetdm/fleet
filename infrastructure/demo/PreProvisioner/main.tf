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
  name      = "preprovisioner"
  full_name = "${var.prefix}-${local.name}"
}

resource "aws_cloudwatch_log_group" "main" {
  name = local.full_name
}

data "aws_iam_policy_document" "lambda-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::917007347864:user/zwinnerman@fleetdm.com"]
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
    resources = [aws_kms_key.ecr.arn]
  }

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

  ingress {
    description      = "egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

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
            value = "2"
          },
          {
            name  = "QUEUED_INSTANCES"
            value = "2"
          },
          {
            name  = "TF_VAR_redis_address"
            value = "${var.redis_cluster.primary_endpoint_address}:6379"
          },
        ])
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_kms_key" "ecr" {
  deletion_window_in_days = 10
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
  }

  depends_on = [
    local_file.backend-config
  ]
}

resource "aws_cloudwatch_event_rule" "main" {
  name_prefix         = var.prefix
  schedule_expression = "rate(5 minutes)"
  is_enabled          = false
}

#resource "aws_cloudwatch_event_target" "main" {
#  rule = aws_cloudwatch_event_rule.main.name
#  arn  = aws_lambda_function.main.arn
#}

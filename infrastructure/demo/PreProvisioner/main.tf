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

data "aws_iam_policy_document" "lambda-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.lambda.id
  policy_arn = aws_iam_policy.lambda.arn
}

resource "aws_iam_role_policy_attachment" "lambda-vpc" {
  role       = aws_iam_role.lambda.id
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
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
  name = "${var.prefix}-preprovisioner"

  assume_role_policy = data.aws_iam_policy_document.lambda-assume-role.json
}

resource "aws_security_group" "lambda" {
  name        = "${var.prefix}-preprovisioner"
  description = "security group for ${var.prefix}-preprovisioner"
  vpc_id      = var.vpc_id

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

resource "aws_lambda_function" "main" {
  image_uri                      = docker_registry_image.main.name
  package_type                   = "Image"
  function_name                  = "${var.prefix}-preprovisioner"
  role                           = aws_iam_role.lambda.arn
  reserved_concurrent_executions = -1
  timeout                        = 600
  memory_size                    = 512
  vpc_config {
    security_group_ids = [aws_security_group.lambda.id]
    subnet_ids         = var.private_subnets
  }
  environment {
    variables = {
      DYNAMODB_LIFECYCLE_TABLE = var.dynamodb_table.id
      MAX_INSTANCES            = 2
      QUEUED_INSTANCES         = 2

      TF_VAR_mysql_secret       = var.mysql_secret.id
      TF_VAR_mysql_cluster_name = var.eks_cluster.eks_cluster_id
      TF_VAR_cluster_endpoint   = data.aws_eks_cluster.cluster.endpoint
      TF_VAR_cluster_ca_cert    = data.aws_eks_cluster.cluster.certificate_authority.0.data
      TF_VAR_eks_cluster        = var.eks_cluster.eks_cluster_id
    }
  }
  tracing_config {
    mode = "Active"
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

resource "aws_cloudwatch_event_target" "main" {
  rule = aws_cloudwatch_event_rule.main.name
  arn  = aws_lambda_function.main.arn
}

resource "aws_lambda_permission" "main" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.main.id
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.main.arn
}

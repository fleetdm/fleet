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
}

resource "aws_iam_role" "lambda" {
  name = "${var.prefix}-lambda"

  assume_role_policy = data.aws_iam_policy_document.lambda-assume-role.json
}

resource "aws_security_group" "lambda" {
  name   = "${var.prefix}-lambda"
  vpc_id = var.vpc_id

  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_lambda_function" "main" {
  image_uri     = docker_registry_image.main.name
  package_type  = "Image"
  function_name = "${var.prefix}-lambda"
  role          = aws_iam_role.lambda.arn
  vpc_config {
    security_group_ids = [aws_security_group.lambda.id]
    subnet_ids         = var.private_subnets
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

resource "random_uuid" "main" {}

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
}

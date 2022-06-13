resource "aws_lb_listener_rule" "jitprovisioner" {
  listener_arn = var.alb_listener.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.jitprovisioner.arn
  }

  condition {
    host_header {
      values = [var.base_domain]
    }
  }
}

resource "aws_lb_target_group_attachment" "jitprovisioner" {
  target_group_arn = aws_lb_target_group.jitprovisioner.arn
  target_id        = aws_lambda_function.jitprovisioner.arn
  depends_on       = [aws_lambda_permission.jitprovisioner]
}

resource "aws_lambda_permission" "jitprovisioner" {
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.jitprovisioner.arn
  principal     = "elasticloadbalancing.amazonaws.com"
  source_arn    = aws_lb_target_group.jitprovisioner.arn
}

resource "aws_lb_target_group" "jitprovisioner" {
  name                               = "${local.full_name}-lambda"
  target_type                        = "lambda"
  lambda_multi_value_headers_enabled = true
}

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "jitprovisioner" {
  name               = "${var.prefix}-lambda"
  assume_role_policy = data.aws_iam_policy_document.lambda_assume_role.json
}

resource "aws_iam_role_policy_attachment" "jitprovisioner-ecr" {
  role       = aws_iam_role.jitprovisioner.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSFargatePodExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "jitprovisioner-vpc" {
  role       = aws_iam_role.jitprovisioner.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole"
}

resource "aws_lambda_function" "jitprovisioner" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  image_uri                      = docker_registry_image.jitprovisioner.name
  package_type                   = "Image"
  function_name                  = "${var.prefix}-lambda"
  role                           = aws_iam_role.jitprovisioner.arn
  reserved_concurrent_executions = -1
  timeout                        = 600
  memory_size                    = 512
  vpc_config {
    security_group_ids = [aws_security_group.jitprovisioner.id]
    subnet_ids         = var.vpc.private_subnets
  }
  tracing_config {
    mode = "Active"
  }
  environment {
    variables = {
      DYNAMODB_LIFECYCLE_TABLE = "test"
      LIFECYCLE_SFN            = "test"
      FLEET_BASE_URL           = "test"
    }
  }
}

resource "random_uuid" "jitprovisioner" {
  keepers = {
    lambda = data.archive_file.jitprovisioner.output_sha
  }
}

data "archive_file" "jitprovisioner" {
  type        = "zip"
  output_path = "${path.module}/.jitprovisioner.zip"
  source_dir  = "${path.module}/lambda"
}

resource "docker_registry_image" "jitprovisioner" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.jitprovisioner.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/lambda/"
    pull_parent = true
  }
}

resource "aws_security_group" "jitprovisioner" {
  name   = "${var.prefix}-lambda"
  vpc_id = var.vpc.vpc_id
}

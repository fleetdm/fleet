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

resource "aws_iam_policy" "jitprovisioner" {
  name   = "${var.prefix}-jitprovisioner"
  policy = data.aws_iam_policy_document.jitprovisioner.json
}

data "aws_iam_policy_document" "jitprovisioner" {
  statement {
    actions = [
      "dynamodb:BatchGetItem",
      "dynamodb:BatchWriteItem",
      "dynamodb:ConditionCheckItem",
      "dynamodb:PutItem",
      "dynamodb:DescribeTable",
      "dynamodb:DeleteItem",
      "dynamodb:GetItem",
      "dynamodb:Scan",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
    ]
    resources = [var.dynamodb_table.arn, "${var.dynamodb_table.arn}/*"]
  }

  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [var.kms_key.arn, var.mysql_secret_kms.arn]
  }

  statement {
    actions   = ["states:StartExecution"]
    resources = [aws_sfn_state_machine.main.arn]
  }

  statement {
    actions   = ["states:DescribeExecution"]
    resources = ["*"]
  }

  statement {
    actions = [
      "secretsmanager:GetResourcePolicy",
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret",
      "secretsmanager:ListSecretVersionIds"
    ]
    resources = [var.mysql_secret.arn]
  }

  statement {
    actions   = ["secretsmanager:ListSecrets"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy_attachment" "jitprovisioner" {
  role       = aws_iam_role.jitprovisioner.name
  policy_arn = aws_iam_policy.jitprovisioner.arn
}

resource "aws_lambda_function" "jitprovisioner" {
  # If the file is not in the current working directory you will need to include a
  # path.module in the filename.
  image_uri                      = docker_registry_image.jitprovisioner.name
  package_type                   = "Image"
  function_name                  = "${var.prefix}-lambda"
  role                           = aws_iam_role.jitprovisioner.arn
  reserved_concurrent_executions = -1
  kms_key_arn                    = var.kms_key.arn
  timeout                        = 5
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
      DYNAMODB_LIFECYCLE_TABLE = var.dynamodb_table.id
      LIFECYCLE_SFN            = aws_sfn_state_machine.main.arn
      FLEET_BASE_URL           = "${var.base_domain}"
      AUTHORIZATION_PSK        = random_password.authorization.result
      MYSQL_SECRET             = var.mysql_secret.arn
    }
  }
}

module "jitprovisioner-lambda-warmer" {
  source        = "Nuagic/lambda-warmer/aws"
  version       = "3.0.1"
  function_name = aws_lambda_function.jitprovisioner.function_name
  function_arn  = aws_lambda_function.jitprovisioner.arn
# This just needs to have a request to parse.
  input         = <<EOINPUT
{
    "requestContext": {
        "elb": {
            "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-2:123456789012:targetgroup/lambda-279XGJDqGZ5rsrHC2Fjr/49e9d65c45c6791a"
        }
    },
    "httpMethod": "GET",
    "path": "/health",
    "queryStringParameters": {
        "query": "1234ABCD"
    },
    "multiValueHeaders": {
        "accept": ["text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8"],
        "accept-encoding": ["gzip"],
        "accept-language": ["en-US,en;q=0.9"],
        "connection": ["keep-alive"],
        "host": ["lambda-alb-123578498.us-east-2.elb.amazonaws.com"],
        "upgrade-insecure-requests": ["1"],
        "user-agent": ["Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36"],
        "x-amzn-trace-id": ["Root=1-5c536348-3d683b8b04734faae651f476"],
        "x-forwarded-for": ["72.12.164.125"],
        "x-forwarded-port": ["80"],
        "x-forwarded-proto": ["http"],
        "x-imforwards": ["20"]
    },
    "body": "",
    "isBase64Encoded": false
}
EOINPUT
}

resource "random_password" "authorization" {
  length  = 16
  special = false
}

output "jitprovisioner" {
  value = aws_lambda_function.jitprovisioner
}

resource "random_uuid" "jitprovisioner" {
  keepers = {
    lambda = data.archive_file.jitprovisioner.output_sha
  }
}

# Use the local to make the trigger work.
locals {
  fleet_tag = "v4.34.1"
}

resource "null_resource" "standard-query-library" {
  triggers = {
    # Trick this to run if the file doesn't exist or if tag changes.
    # In the case it doesn't exist, this will say it needs to apply twice,
    # so not truly idempotent, but as close as null_resource allows.
    file_exists = fileexists("${path.module}/lambda/standard-query-library.yml") ? local.fleet_tag : timestamp()
  }

  provisioner "local-exec" {
    working_dir = "${path.module}/../../../"
    command     = "git archive fleet-${local.fleet_tag} docs/01-Using-Fleet/standard-query-library/standard-query-library.yml | tar -xO docs/01-Using-Fleet/standard-query-library/standard-query-library.yml > infrastructure/sandbox/JITProvisioner/lambda/standard-query-library.yml"
  }
}

data "archive_file" "jitprovisioner" {
  type        = "zip"
  output_path = "${path.module}/.jitprovisioner.zip"
  source_dir  = "${path.module}/lambda"
  depends_on = [
    null_resource.standard-query-library
  ]
}

resource "docker_registry_image" "jitprovisioner" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.jitprovisioner.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/lambda/"
    pull_parent = true
    platform    = "linux/amd64"
  }
  depends_on = [
    null_resource.standard-query-library
  ]
}

resource "aws_security_group" "jitprovisioner" {
  name        = local.full_name
  vpc_id      = var.vpc.vpc_id
  description = local.full_name
  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "${var.prefix}-iam_for_lambda"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_lambda_permission" "allow_bucket" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.func.arn
  principal     = "s3.amazonaws.com"
  source_arn    = var.access_logs_s3_bucket.s3_bucket_arn
}

resource "aws_lambda_function" "func" {
  filename      = data.archive_file.lambda.output_path
  function_name = "${var.prefix}-lambda"
  role          = aws_iam_role.iam_for_lambda.arn
  handler       = "main.lambda_handler"
  runtime       = "python3.8"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = var.access_logs_s3_bucket.s3_bucket_id

  lambda_function {
    lambda_function_arn = aws_lambda_function.func.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "AWSLogs/"
    filter_suffix       = ".log"
  }

  depends_on = [aws_lambda_permission.allow_bucket]
}

data "archive_file" "lambda" {
  type        = "zip"
  source_file = "${path.module}/main.py"
  output_path = "${path.module}/lambda.zip"
}

resource "aws_iam_service_linked_role" "main" {
  aws_service_name = "opensearchservice.amazonaws.com"
}

resource "aws_security_group" "os" {
  name   = var.prefix
  vpc_id = var.vpc.vpc_id

  ingress {
    from_port = 443
    to_port   = 443
    protocol  = "tcp"

    cidr_blocks = [
      "10.0.0.0/8" # TODO: vpn and lambda SG only
    ]
  }
}

resource "aws_opensearch_domain" "main" {
  domain_name    = var.prefix
  engine_version = "OpenSearch_1.3"

  cluster_config {
    instance_type          = "t3.small.search"
    instance_count         = 1
    zone_awareness_enabled = true
  }

  vpc_options {
    subnet_ids = var.vpc.private_subnets

    security_group_ids = [aws_security_group.os.id]
  }

  advanced_options = {
    "rest.action.multi.allow_explicit_index" = "true"
  }

  access_policies = <<CONFIG
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "es:*",
            "Principal": "*",
            "Effect": "Allow",
            "Resource": "arn:aws:es:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:domain/${var.prefix}/*"
        }
    ]
}
CONFIG

  depends_on = [aws_iam_service_linked_role.main]
}

data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

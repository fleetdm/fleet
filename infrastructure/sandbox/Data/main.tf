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

resource "aws_iam_policy" "lambda_logging" {
  name        = "${var.prefix}-lambda_logging"
  path        = "/"
  description = "IAM policy for logging from a lambda"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*",
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = aws_iam_policy.lambda_logging.arn
}

resource "aws_iam_policy" "lambda" {
  name   = "${var.prefix}-lambda"
  policy = data.aws_iam_policy_document.lambda.json
}

data "aws_iam_policy_document" "lambda" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${var.access_logs_s3_bucket.s3_bucket_arn}/*"]
  }

  statement {
    actions   = ["kms:Decrypt"]
    resources = [var.kms_key.arn]
  }

  statement {
    actions   = ["ec2:*NetworkInterface*"]
    resources = ["*"]
  }
}

resource "aws_iam_role_policy_attachment" "lambda" {
  role       = aws_iam_role.iam_for_lambda.name
  policy_arn = aws_iam_policy.lambda.arn
}

resource "aws_lambda_permission" "allow_bucket" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.func.arn
  principal     = "s3.amazonaws.com"
  source_arn    = var.access_logs_s3_bucket.s3_bucket_arn
}

resource "aws_lambda_function" "func" {
  filename         = data.archive_file.lambda.output_path
  source_code_hash = filebase64sha256("${path.module}/lambda.zip")
  function_name    = "${var.prefix}-lambda"
  role             = aws_iam_role.iam_for_lambda.arn
  handler          = "main.lambda_handler"
  runtime          = "python3.8"
  timeout          = 30
  vpc_config {
    subnet_ids         = var.vpc.private_subnets
    security_group_ids = [aws_security_group.sg_for_lambda.id]
  }
  environment {
    variables = {
      ES_URL = "http://${aws_opensearch_domain.main.endpoint}:80"
    }
  }
}

resource "aws_security_group" "sg_for_lambda" {
  name   = "${var.prefix}-lambda"
  vpc_id = var.vpc.vpc_id

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"

    cidr_blocks = [
      "0.0.0.0/0" # TODO: vpn and lambda SG only
    ]
  }
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = var.access_logs_s3_bucket.s3_bucket_id

  lambda_function {
    lambda_function_arn = aws_lambda_function.func.arn
    events              = ["s3:ObjectCreated:*"]
    filter_suffix       = ".log.gz"
  }

  depends_on = [aws_lambda_permission.allow_bucket]
}

data "archive_file" "lambda" {
  type        = "zip"
  source_dir  = "${path.module}/lambda"
  output_path = "${path.module}/lambda.zip"
}

resource "aws_security_group" "os" {
  name   = var.prefix
  vpc_id = var.vpc.vpc_id

  ingress {
    from_port = 80
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
    zone_awareness_enabled = false
  }

  vpc_options {
    subnet_ids = [var.vpc.private_subnets[0]]

    security_group_ids = [aws_security_group.os.id]
  }

  advanced_options = {
    "rest.action.multi.allow_explicit_index" = "true"
  }

  ebs_options {
    ebs_enabled = true
    volume_size = 10
    volume_type = "gp2"
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
}

data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

resource "aws_iam_service_linked_role" "main" {
  aws_service_name = "opensearchservice.amazonaws.com"
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id       = var.vpc.vpc_id
  service_name = "com.amazonaws.us-east-2.s3"
}

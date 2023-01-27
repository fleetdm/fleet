data "aws_region" "current" {}
data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "osquery_firehose_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["firehose.amazonaws.com"]
      type        = "Service"
    }
  }
}

data "aws_iam_policy_document" "firehose_policy" {
  statement {
    effect = "Allow"
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject",
      "s3:PutObjectAcl" // required according to https://docs.aws.amazon.com/firehose/latest/dev/controlling-access.html#using-iam-s3
    ]
    resources = [
      "arn:aws:s3:::${var.results_destination_s3_bucket}",
      "arn:aws:s3:::${var.results_destination_s3_bucket}/*",
      "arn:aws:s3:::${var.status_destination_s3_bucket}",
      "arn:aws:s3:::${var.status_destination_s3_bucket}/*"
    ]
  }

  statement {
    effect    = "Allow"
    actions   = ["kms:GenerateDataKey*"]
    resources = [var.kms_key_arn]
  }

  statement {
    effect    = "Allow"
    actions   = ["logs:PutLogEvents"]
    resources = [
      "arn:aws:logs:${data.aws_region.current.id}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.firehose_results_name}:*",
      "arn:aws:logs:${data.aws_region.current.id}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.firehose_status_name}:*"
    ]
  }
}

resource "aws_iam_role" "firehose" {
  name = "${var.customer_prefix}-firehose"
  assume_role_policy = data.aws_iam_policy_document.osquery_firehose_assume_role.json
}

resource "aws_iam_policy" "firehose" {
  policy = data.aws_iam_policy_document.firehose_policy.json
}

resource "aws_iam_role_policy_attachment" "firehose" {
  policy_arn = aws_iam_policy.firehose.arn
  role       = aws_iam_role.firehose.name
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_results" {
  name        = var.firehose_results_name
  destination = "s3"

  s3_configuration {
    prefix      = var.results_object_prefix
    role_arn    = aws_iam_role.firehose.arn
    bucket_arn  = "arn:aws:s3:::${var.results_destination_s3_bucket}"
    kms_key_arn = var.kms_key_arn
  }
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_status" {
  name        = var.firehose_status_name
  destination = "s3"

  s3_configuration {
    prefix      = var.status_object_prefix
    role_arn    = aws_iam_role.firehose
    bucket_arn  = "arn:aws:s3:::${var.status_destination_s3_bucket}"
    kms_key_arn = var.kms_key_arn
  }
}

data "aws_iam_policy_document" "firehose-logging" {
  statement {
    actions = [
      "firehose:DescribeDeliveryStream",
      "firehose:PutRecord",
      "firehose:PutRecordBatch",
    ]
    resources = [aws_kinesis_firehose_delivery_stream.osquery_results.arn, aws_kinesis_firehose_delivery_stream.osquery_status.arn]
  }
}

resource "aws_iam_policy" "firehose-logging" {
  name        = "fleet-firehose-logging"
  description = "An IAM policy for fleet to log to Firehose destinations"
  policy      = data.aws_iam_policy_document.firehose-logging.json
}

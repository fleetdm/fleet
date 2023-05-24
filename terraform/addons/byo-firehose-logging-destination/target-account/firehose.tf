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
      aws_s3_bucket.destination.arn,
      "${aws_s3_bucket.destination.arn}/*",
    ]
  }

  statement {
    effect  = "Allow"
    actions = ["logs:PutLogEvents"]
    resources = join([
      "arn:aws:logs:${data.aws_region.current.id}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.firehose_results_name}:*",
      "arn:aws:logs:${data.aws_region.current.id}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.firehose_status_name}:*",
      ],
      var.firehose_status_name == "" ? [] : [
        "arn:aws:logs:${data.aws_region.current.id}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.firehose_audit_name}:*"
    ])
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey"
    ]
    resources = [data.aws_kms_alias.s3.arn]
  }

}

resource "aws_iam_role" "firehose" {
  assume_role_policy = data.aws_iam_policy_document.osquery_firehose_assume_role.json
}

resource "aws_iam_policy" "firehose" {
  policy = data.aws_iam_policy_document.firehose_policy.json
}

resource "aws_iam_role_policy_attachment" "firehose" {
  policy_arn = aws_iam_policy.firehose.arn
  role       = aws_iam_role.firehose.name
}

resource "aws_kms_key" "firehose" {
  enable_key_rotation = true
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_results" {
  name        = var.firehose_results_name
  destination = "s3"

  server_side_encryption {
    key_arn = aws_kms_key.firehose.arn
  }

  s3_configuration {
    prefix     = var.results_prefix
    role_arn   = aws_iam_role.firehose.arn
    bucket_arn = aws_s3_bucket.destination.arn
  }
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_status" {
  name        = var.firehose_status_name
  destination = "s3"

  server_side_encryption {
    key_arn = aws_kms_key.firehose.arn
  }

  s3_configuration {
    prefix     = var.status_prefix
    role_arn   = aws_iam_role.firehose.arn
    bucket_arn = aws_s3_bucket.destination.arn
  }
}

resource "aws_kinesis_firehose_delivery_stream" "fleet_audit" {
  count       = length(var.firehose_audit_name) > 0 ? 1 : 0
  name        = var.firehose_audit_name
  destination = "s3"

  server_side_encryption {
    key_arn = aws_kms_key.firehose.arn
  }

  s3_configuration {
    prefix     = var.audit_prefix
    role_arn   = aws_iam_role.firehose.arn
    bucket_arn = aws_s3_bucket.destination.arn
  }
}

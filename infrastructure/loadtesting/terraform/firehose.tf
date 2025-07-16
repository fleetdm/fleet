resource "aws_s3_bucket" "osquery-results" { #tfsec:ignore:aws-s3-encryption-customer-key tfsec:ignore:aws-s3-enable-bucket-logging tfsec:ignore:aws-s3-enable-versioning
  bucket = "${local.prefix}-loadtest-osquery-logs-archive"
  
  # Allow destroy of non-empty buckets
  force_destroy = true

  #checkov:skip=CKV_AWS_18:dev env
  #checkov:skip=CKV_AWS_144:dev env
  #checkov:skip=CKV_AWS_21:dev env
}

resource "aws_s3_bucket_server_side_encryption_configuration" "osquery-results" {
  bucket = aws_s3_bucket.osquery-results.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "osquery-results" {
  bucket = aws_s3_bucket.osquery-results.id

  rule {
    id     = "rule-1"
    status = "Enabled"
    expiration {
      days = 1
    }
  }
}

resource "aws_s3_bucket_public_access_block" "osquery-results" {
  bucket = aws_s3_bucket.osquery-results.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket" "osquery-status" { #tfsec:ignore:aws-s3-encryption-customer-key tfsec:ignore:aws-s3-enable-bucket-logging tfsec:ignore:aws-s3-enable-versioning
  bucket = "${local.prefix}-loadtest-osquery-status-archive"
  
  # Allow destroy of non-empty buckets
  force_destroy = true

  #checkov:skip=CKV_AWS_18:dev env
  #checkov:skip=CKV_AWS_144:dev env
  #checkov:skip=CKV_AWS_21:dev env
}

resource "aws_s3_bucket_lifecycle_configuration" "osquery-status" {
  bucket = aws_s3_bucket.osquery-status.id

  rule {
    id     = "rule-1"
    status = "Enabled"
    expiration {
      days = 1
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "osquery-status" {
  bucket = aws_s3_bucket.osquery-status.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "osquery-status" {
  bucket = aws_s3_bucket.osquery-status.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

data "aws_iam_policy_document" "osquery_results_policy_doc" {
  statement {
    effect = "Allow"
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject"
    ]
    resources = [aws_s3_bucket.osquery-results.arn, "${aws_s3_bucket.osquery-results.arn}/*"] #tfsec:ignore:aws-iam-no-policy-wildcards
  }
}

data "aws_iam_policy_document" "osquery_status_policy_doc" {
  statement {
    effect = "Allow"
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject"
    ]
    resources = [aws_s3_bucket.osquery-status.arn, "${aws_s3_bucket.osquery-status.arn}/*"] #tfsec:ignore:aws-iam-no-policy-wildcards
  }
}

resource "aws_iam_policy" "firehose-results" {
  name   = "${local.prefix}-osquery_results_firehose_policy"
  policy = data.aws_iam_policy_document.osquery_results_policy_doc.json
}

resource "aws_iam_policy" "firehose-status" {
  name   = "${local.prefix}-osquery_status_firehose_policy"
  policy = data.aws_iam_policy_document.osquery_status_policy_doc.json
}

resource "aws_iam_role" "firehose-results" {
  assume_role_policy = data.aws_iam_policy_document.osquery_firehose_assume_role.json
}

resource "aws_iam_role" "firehose-status" {
  assume_role_policy = data.aws_iam_policy_document.osquery_firehose_assume_role.json
}

resource "aws_iam_role_policy_attachment" "firehose-results" {
  policy_arn = aws_iam_policy.firehose-results.arn
  role       = aws_iam_role.firehose-results.name
}

resource "aws_iam_role_policy_attachment" "firehose-status" {
  policy_arn = aws_iam_policy.firehose-status.arn
  role       = aws_iam_role.firehose-status.name
}

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

resource "aws_kinesis_firehose_delivery_stream" "osquery_results" {
  name        = "${local.prefix}-osquery_results"
  destination = "s3"

  s3_configuration {
    role_arn   = aws_iam_role.firehose-results.arn
    bucket_arn = aws_s3_bucket.osquery-results.arn
  }
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_status" {
  name        = "${local.prefix}-osquery_status"
  destination = "s3"

  s3_configuration {
    role_arn   = aws_iam_role.firehose-status.arn
    bucket_arn = aws_s3_bucket.osquery-status.arn
  }
}

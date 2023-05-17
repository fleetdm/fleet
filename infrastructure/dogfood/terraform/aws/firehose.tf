// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future. 
// No versioning on this bucket is by design.
// Bucket logging is not supported in our Fleet Terraforms at the moment. It can be enabled by the
// organizations deploying Fleet, and we will evaluate the possibility of providing this capability
// in the future.
resource "aws_s3_bucket" "osquery-results" { #tfsec:ignore:aws-s3-encryption-customer-key:exp:2022-07-01  #tfsec:ignore:aws-s3-enable-versioning #tfsec:ignore:aws-s3-enable-bucket-logging:exp:2022-06-15
  bucket = var.osquery_results_s3_bucket
  acl    = "private"

  lifecycle_rule {
    enabled = true
    expiration {
      days = 1
    }
  }

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "aws:kms"
      }
    }
  }
}

resource "aws_s3_bucket_public_access_block" "osquery-results" {
  bucket                  = aws_s3_bucket.osquery-results.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future.
// No versioning on this bucket is by design.
// Bucket logging is not supported in our Fleet Terraforms at the moment. It can be enabled by the
// organizations deploying Fleet, and we will evaluate the possibility of providing this capability
// in the future.
resource "aws_s3_bucket" "osquery-status" { #tfsec:ignore:aws-s3-encryption-customer-key:exp:2022-07-01 #tfsec:ignore:aws-s3-enable-versioning #tfsec:ignore:aws-s3-enable-bucket-logging:exp:2022-06-15
  bucket = var.osquery_status_s3_bucket
  acl    = "private"

  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "aws:kms"
      }
    }
  }
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

resource "aws_s3_bucket_public_access_block" "osquery-status" {
  bucket                  = aws_s3_bucket.osquery-status.id
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
    // This bucket is single-purpose and using a wildcard is not problematic
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
    // This bucket is single-purpose and using a wildcard is not problematic
    resources = [aws_s3_bucket.osquery-status.arn, "${aws_s3_bucket.osquery-status.arn}/*"] #tfsec:ignore:aws-iam-no-policy-wildcards
  }
}

resource "aws_iam_policy" "firehose-results" {
  name   = "osquery_results_firehose_policy"
  policy = data.aws_iam_policy_document.osquery_results_policy_doc.json
}

resource "aws_iam_policy" "firehose-status" {
  name   = "osquery_status_firehose_policy"
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
  name        = "osquery_results"
  destination = "s3"

  s3_configuration {
    role_arn   = aws_iam_role.firehose-results.arn
    bucket_arn = aws_s3_bucket.osquery-results.arn
  }
}

resource "aws_kinesis_firehose_delivery_stream" "osquery_status" {
  name        = "osquery_status"
  destination = "s3"

  s3_configuration {
    role_arn   = aws_iam_role.firehose-status.arn
    bucket_arn = aws_s3_bucket.osquery-status.arn
  }
}

resource "aws_s3_bucket" "osquery-results" {
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

resource "aws_s3_bucket" "osquery-status" {
  bucket = var.osquery_status_s3_bucket
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
    resources = [aws_s3_bucket.osquery-results.arn, "${aws_s3_bucket.osquery-results.arn}/*"]
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
    resources = [aws_s3_bucket.osquery-status.arn, "${aws_s3_bucket.osquery-status.arn}/*"]
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
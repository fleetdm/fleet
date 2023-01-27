variable "osquery_results_bucket" {
  type        = string
  description = "name of the bucket to store osquery results logs"
}

variable "osquery_status_bucket" {
  type        = string
  description = "name of the bucket to store osquery status logs"
}

variable "fleet_iam_role_arn" {
  type        = string
  description = "the arn of the fleet role that firehose will assume to write data to your bucket"
}

data "aws_caller_identity" "current" {}

data "aws_iam_policy_document" "results" {
  statement {
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
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
      aws_s3_bucket.osquery-results.arn,
      "${aws_s3_bucket.osquery-results.arn}/*"
    ]
  }

  statement {
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
    effect    = "Allow"
    actions   = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.osquery-results.arn}/*"]
    condition {
      test     = "StringEquals"
      values   = ["bucket-owner-full-control"]
      variable = "s3:x-amz-acl"
    }
  }
}

data "aws_iam_policy_document" "status" {
  statement {
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
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
      aws_s3_bucket.osquery-status.arn,
      "${aws_s3_bucket.osquery-status.arn}/*"
    ]
  }

  statement {
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
    effect    = "Allow"
    actions   = ["s3:PutObject"]
    resources = ["${aws_s3_bucket.osquery-status.arn}/*"]
    condition {
      test     = "StringEquals"
      values   = ["bucket-owner-full-control"]
      variable = "s3:x-amz-acl"
    }
  }
}

resource "aws_s3_bucket" "osquery-results" {
  bucket = var.osquery_results_bucket
}

resource "aws_s3_bucket" "osquery-status" {
  bucket = var.osquery_status_bucket
}

resource "aws_s3_bucket_policy" "results" {
  bucket = aws_s3_bucket.osquery-results.id
  policy = data.aws_iam_policy_document.results.json
}

resource "aws_s3_bucket_policy" "status" {
  bucket = aws_s3_bucket.osquery-status.id
  policy = data.aws_iam_policy_document.status.json
}

resource "aws_s3_bucket_public_access_block" "results" {
  bucket                  = aws_s3_bucket.osquery-results.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_public_access_block" "status" {
  bucket                  = aws_s3_bucket.osquery-status.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}


resource "aws_s3_bucket_acl" "results" {
  bucket = aws_s3_bucket.osquery-results.id
  acl    = "private"
}

resource "aws_s3_bucket_acl" "status" {
  bucket = aws_s3_bucket.osquery-status.id
  acl    = "private"
}

data "aws_iam_policy_document" "key_policy" {
  // self account has access to key
  statement {
    principals {
      identifiers = [
        "arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"
      ]
      type = "AWS"
    }
    effect    = "Allow"
    actions   = ["*"]
    resources = ["*"]
  }

  // only allow the IAM role from fleet aws account
  statement {
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
    effect    = "Allow"
    actions   = ["kms:GenerateDataKey*"]
    resources = ["*"] // this is basically "self" aka this particular key
  }
}

// customer managed key to allow other aws account access
resource "aws_kms_key" "key" {
  enable_key_rotation = true
  policy              = data.aws_iam_policy_document.key_policy.json
  description         = "key used for osquery results and status bucket encryption"
}

// enable server side encryption with KMS key
resource "aws_s3_bucket_server_side_encryption_configuration" "results" {
  bucket = aws_s3_bucket.osquery-results.id
  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.key.id
      sse_algorithm     = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "status" {
  bucket = aws_s3_bucket.osquery-status.id
  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.key.id
      sse_algorithm     = "aws:kms"
    }
  }
}


data "aws_caller_identity" "current" {}

# IAM policy document for the KMS key
data "aws_iam_policy_document" "kms_key_policy" {
  statement {
    sid       = "Enable IAM User Permissions"
    actions   = ["kms:*"]
    resources = ["*"]
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }
  }
}

# Create a KMS key for encrypting the S3 bucket
resource "aws_kms_key" "s3_encryption_key" {
  description = "KMS key for S3 bucket encryption"
  is_enabled  = true
  policy      = data.aws_iam_policy_document.kms_key_policy.json
}

# Create an S3 bucket with server-side encryption using the customer-managed key
resource "aws_s3_bucket" "carve_results_bucket" {
  bucket = var.bucket_name
}

resource "aws_s3_bucket_public_access_block" "carve_results" {
  bucket = aws_s3_bucket.carve_results_bucket.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "sse" {
  bucket = aws_s3_bucket.carve_results_bucket.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3_encryption_key.key_id
    }
  }
}

# Create an IAM policy which allows the necessary S3 actions
resource "aws_iam_policy" "s3_access_policy" {
  name   = "s3_access_policy"
  policy = data.aws_iam_policy_document.s3_policy.json
}

# IAM policy document
data "aws_iam_policy_document" "s3_policy" {
  statement {
    actions = [
      "s3:GetObject*",
      "s3:PutObject*",
      "s3:ListBucket*",
      "s3:ListMultipartUploadParts*",
      "s3:DeleteObject",
      "s3:CreateMultipartUpload",
      "s3:AbortMultipartUpload",
      "s3:ListMultipartUploadParts",
      "s3:GetBucketLocation"
    ]

    resources = [
      aws_s3_bucket.carve_results_bucket.arn,
      "${aws_s3_bucket.carve_results_bucket.arn}/*"
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
      "kms:Encrypt"
    ]
    resources = [aws_kms_key.s3_encryption_key.arn]
  }
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
    dynamic "condition" {
      for_each = length(var.sts_external_id) > 0 ? [1] : []
      content {
        test     = "StringEquals"
        variable = "sts:ExternalId"
        values   = [var.sts_external_id]
      }
    }
  }
}

resource "aws_iam_role" "carve_s3_delegation_role" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

# Attach the policy to the role
resource "aws_iam_role_policy_attachment" "s3_access_attachment" {
  role       = aws_iam_role.carve_s3_delegation_role.name
  policy_arn = aws_iam_policy.s3_access_policy.arn
}

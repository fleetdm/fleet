data "aws_iam_policy_document" "software_installers" {
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
    resources = [aws_s3_bucket.software_installers.arn, "${aws_s3_bucket.software_installers.arn}/*"]
  }
  dynamic "statement" {
    for_each = local.software_installers_kms_policy
    content {
      sid       = try(statement.value.sid, "")
      actions   = try(statement.value.actions, [])
      resources = try(statement.value.resources, [])
      effect    = try(statement.value.effect, null)
      dynamic "principals" {
        for_each = try(statement.value.principals, [])
        content {
          type        = principals.value.type
          identifiers = principals.value.identifiers
        }
      }
      dynamic "condition" {
        for_each = try(statement.value.conditions, [])
        content {
          test     = condition.value.test
          variable = condition.value.variable
          values   = condition.value.values
        }
      }
    }
  }
}

resource "aws_iam_policy" "software_installers" {
  policy = data.aws_iam_policy_document.software_installers.json
}

resource "aws_iam_role_policy_attachment" "software_installers" {
  policy_arn = aws_iam_policy.software_installers.arn
  role       = aws_iam_role.main.name
}

resource "aws_s3_bucket" "software_installers" { #tfsec:ignore:aws-s3-encryption-customer-key:exp:2022-07-01  #tfsec:ignore:aws-s3-enable-versioning #tfsec:ignore:aws-s3-enable-bucket-logging:exp:2022-06-15
  bucket_prefix = terraform.workspace
  
  # Allow destroy of non-empty buckets
  force_destroy = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "software_installers" {
  bucket = aws_s3_bucket.software_installers.bucket
  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = aws_kms_key.software_installers.id
      sse_algorithm = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "software_installers" {
  bucket                  = aws_s3_bucket.software_installers.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_kms_key" "software_installers" {
  enable_key_rotation = true
}

resource "aws_kms_alias" "software_installers" {
  target_key_id = aws_kms_key.software_installers.id
  name          = "alias/${terraform.workspace}-software-installers"
}
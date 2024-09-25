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
}

resource "aws_s3_bucket_server_side_encryption_configuration" "software_installers" {
  bucket = aws_s3_bucket.software_installers.bucket
  rule {
    apply_server_side_encryption_by_default {
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

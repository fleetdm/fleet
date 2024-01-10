data "aws_region" "current" {}
data "aws_caller_identity" "current" {}
data "aws_kms_alias" "s3" {
  name = "alias/aws/s3"
}

resource "aws_s3_bucket" "destination" {
  bucket = var.osquery_logging_destination_bucket_name
}

resource "aws_s3_bucket_public_access_block" "destination" {
  bucket                  = aws_s3_bucket.destination.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

// Objects in S3 are now encrypted by default https://aws.amazon.com/blogs/aws/amazon-s3-encrypts-new-objects-by-default/
// If you need more granular control, use a customer managed KMS Key
resource "aws_s3_bucket_server_side_encryption_configuration" "destination" {
  bucket = aws_s3_bucket.destination.id
  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
  }
}

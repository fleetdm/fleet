// file carving destination with versioning disabled and lifecycle to ensure files get deleted and
// no version is ever kept

// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future.
// Bucket logging is not supported in our Fleet Terraforms at the moment. It can be enabled by the
// organizations deploying Fleet, and we will evaluate the possibility of providing this capability
// in the future.
resource "aws_s3_bucket" "osquery-carve" { #tfsec:ignore:aws-s3-enable-versioning #tfsec:ignore:aws-s3-encryption-customer-key:exp:2022-07-01 #tfsec:ignore:aws-s3-enable-bucket-logging:exp:2022-06-15
  bucket = "osquery-carve-${terraform.workspace}"
  acl    = "private"

  lifecycle_rule {
    enabled = true
    expiration {
      days = 7
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

resource "aws_s3_bucket_public_access_block" "osquery-carve" {
  bucket                  = aws_s3_bucket.osquery-carve.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}
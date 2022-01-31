// file carving destination
resource "aws_s3_bucket" "osquery-carve" {
  bucket = "${var.prefix}fleet-osquery-carve"
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
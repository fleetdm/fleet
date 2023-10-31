resource "aws_s3_bucket" "installers" {
  bucket = "${var.prefix}-installers"
}

resource "aws_s3_bucket_public_access_block" "installers" {
  bucket = aws_s3_bucket.installers.id

  block_public_acls   = true
  block_public_policy = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "installers" {
  bucket = aws_s3_bucket.installers.id

  rule {
    apply_server_side_encryption_by_default {
      kms_master_key_id = var.kms_key.arn
      sse_algorithm     = "aws:kms"
    }
  }
}

output "installer_bucket" {
  value = aws_s3_bucket.installers
}

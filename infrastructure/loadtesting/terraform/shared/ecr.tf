resource "aws_kms_key" "main" {
  description             = "${local.prefix}-${random_pet.main.id}"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "prometheus-to-cloudwatch" {
  name                 = "prometheus-to-cloudwatch"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }
}

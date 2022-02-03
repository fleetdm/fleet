resource "aws_ecr_repository" "prometheus-to-cloudwatch" {
  name                 = "prometheus-to-cloudwatch"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

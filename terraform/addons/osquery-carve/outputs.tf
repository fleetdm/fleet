output "fleet_extra_environment_variables" {
  value = {
    FLEET_S3_CARVES_BUCKET = aws_s3_bucket.main.bucket
    FLEET_S3_CARVES_PREFIX = "carve_results/"
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.main.arn
  ]
}

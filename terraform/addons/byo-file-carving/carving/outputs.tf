output "fleet_extra_environment_variables" {
  value = {
    FLEET_S3_STS_ASSUME_ROLE_ARN = var.iam_role_arn
    FLEET_S3_STS_EXTERNAL_ID     = var.sts_external_id
    FLEET_S3_BUCKET              = var.s3_bucket_name
    FLEET_S3_REGION              = var.s3_bucket_region
    FLEET_S3_PREFIX              = var.s3_carve_prefix
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.fleet-assume-role.arn
  ]
}

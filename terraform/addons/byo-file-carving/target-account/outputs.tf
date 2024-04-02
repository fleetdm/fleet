output "iam_role_arn" {
  value = aws_iam_role.carve_s3_delegation_role.arn
}

output "s3_bucket_name" {
  value = aws_s3_bucket.carve_results_bucket.id
}

output "s3_bucket_region" {
  value = aws_s3_bucket.carve_results_bucket.region
}
output "kms_key_arn" {
  value = aws_kms_key.key.arn
}

output "results_bucket_name" {
  value = aws_s3_bucket.osquery-results.id
}

output "status_bucket_name" {
  value = aws_s3_bucket.osquery-status.id
}
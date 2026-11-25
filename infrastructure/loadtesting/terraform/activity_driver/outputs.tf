output "stop_marker" {
  description = "The workload pauses while this object exists; a trip creates it. Remove it to resume."
  value       = "s3://${aws_s3_bucket.driver.id}/STOP"
}

output "bucket" {
  value = aws_s3_bucket.driver.id
}

output "log_group" {
  value = aws_cloudwatch_log_group.driver.name
}

output "secrets" {
  description = "Secrets whose values are set by hand (see README)."
  value = {
    token   = aws_secretsmanager_secret.token.name
    profile = var.with_profile ? aws_secretsmanager_secret.profile[0].name : null
    notify  = var.notify ? aws_secretsmanager_secret.notify[0].name : null
  }
}

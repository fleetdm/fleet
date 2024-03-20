output "extra_execution_iam_policies" {
  value = [
    aws_iam_policy.main.arn
  ]
}

output "secret_arn" {
  value = var.secret_arn
}
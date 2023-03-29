output "fleet_extra_environment_variables" {
  value = {
    FLEET_EMAIL_BACKEND = "ses"
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.main.arn
  ]
}

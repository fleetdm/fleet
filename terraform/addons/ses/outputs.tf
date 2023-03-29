output "fleet_extra_environment_variables" {
  value = {
    FLEET_EMAIL_BACKEND  = "ses"
    FLEET_SES_SOURCE_ARN = aws_ses_domain_identity.default.arn
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.main.arn
  ]
}

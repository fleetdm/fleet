output "fleet_extra_execution_policies" {
  value = [
    aws_iam_policy.saml_auth_proxy.arn
  ]
}

output "name" {
  value = "${var.customer_prefix}-saml-auth-proxy"
}

output "lb" {
  value = module.saml_auth_proxy_alb
}

output "secretsmanager_secret_id" {
  value = aws_secretsmanager_secret.saml_auth_proxy_cert.id
}

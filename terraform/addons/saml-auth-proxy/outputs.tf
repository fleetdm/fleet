output "fleet_extra_execution_policies" {
  value = [
    aws_iam_policy.saml_auth_proxy.arn
  ]
}

output "name" {
  value = "${var.customer_prefix}-saml-auth-proxy"
}

# Keep for legacy support for now
output "lb_target_group_arn" {
  value = module.saml_auth_proxy_alb.target_group_arns[0]
}

output "lb" {
  value = module.saml_auth_proxy_alb
}

output "lb_security_group" {
  value = aws_security_group.saml_auth_proxy_alb.id
}

output "secretsmanager_secret_id" {
  value = aws_secretsmanager_secret.saml_auth_proxy_cert.id
}

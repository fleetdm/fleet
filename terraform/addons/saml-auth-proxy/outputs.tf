output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.saml_auth_proxy.arn
  ]
}

output "name" {
  value = "${var.customer_prefix}-saml-auth-proxy"
}

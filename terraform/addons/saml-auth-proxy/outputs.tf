output "fleet_extra_execution_policies" {
  value = [
    aws_iam_policy.saml_auth_proxy.arn
  ]
}

output "name" {
  value = "${var.customer_prefix}-saml-auth-proxy"
}

output "lb_target_group_arn" {
  value = module.saml_auth_proxy_alb.target_group_arns[0]
}

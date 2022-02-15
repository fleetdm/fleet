output "nameservers_fleetctl" {
  value = aws_route53_zone.dogfood_fleetctl_com.name_servers
}

output "nameservers_fleetdm" {
  value = aws_route53_zone.dogfood_fleetdm_com.name_servers
}

output "backend_security_group" {
  value = aws_security_group.backend.arn
}

output "private_subnets" {
  value = module.vpc.private_subnet_arns
}
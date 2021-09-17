output "nameservers_fleetctl" {
  value = aws_route53_zone.dogfood_fleetctl_com.name_servers
}

output "nameservers_fleetdm" {
  value = aws_route53_zone.dogfood_fleetdm_com.name_servers
}


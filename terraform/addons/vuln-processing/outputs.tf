output "fleet_extra_environment_variables" {
  value = {
    FLEET_VULNERABILITIES_DISABLE_SCHEDULE = "true"
  }
}

output "enable_dns_hostnames" {
  value = true
}
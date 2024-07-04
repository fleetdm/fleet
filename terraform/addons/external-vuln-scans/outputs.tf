output "extra_environment_variables" {
  value = {
    FLEET_VULNERABILITIES_DISABLE_SCHEDULE = "true"
  }
}

output "vuln_service_arn" {
  value = aws_ecs_service.fleet.id
}

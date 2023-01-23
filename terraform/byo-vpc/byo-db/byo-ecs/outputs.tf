output "service" {
  value = aws_ecs_service.fleet
}

output "task_definition" {
  value = aws_ecs_task_definition.backend
}

output "non_circular" {
  value = {
    "security_groups" = var.fleet_config.networking.security_groups == null ? aws_security_group.main.*.id : var.fleet_config.networking.security_groups,
    "subnets"         = var.fleet_config.networking.subnets,
  }
}

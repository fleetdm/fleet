output "service" {
  value = aws_ecs_service.fleet
}

output "task_definition" {
  value = aws_ecs_task_definition.backend
}

output "iam_role_arn" {
  # Always respond sanely even if we did not generate
  value = var.fleet_config.iam_role_arn == null ? aws_iam_role.main[0].arn : var.fleet_config.iam_role_arn
}

output "execution_iam_role_arn" {
  value = aws_iam_role.execution.arn
}

output "logging_config" {
  # Always respond sanely even if we did not generate
  value = {
    awslogs-group         = var.fleet_config.awslogs.create == true ? aws_cloudwatch_log_group.main[0].name : var.fleet_config.awslogs.name
    awslogs-region        = var.fleet_config.awslogs.create == true ? data.aws_region.current.name : var.fleet_config.awslogs.region
    awslogs-stream-prefix = var.fleet_config.awslogs.prefix
  }
}

output "non_circular" {
  value = {
    "security_groups" = var.fleet_config.networking.security_groups == null ? aws_security_group.main.*.id : var.fleet_config.networking.security_groups,
    "subnets"         = var.fleet_config.networking.subnets,
  }
}

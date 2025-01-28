output "service" {
  value = aws_ecs_service.fleet
}

output "appautoscaling_target" {
  value = aws_appautoscaling_target.ecs_target
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

output "fleet_config" {
  value = var.fleet_config
}

output "fleet_server_private_key_secret_arn" {
  value = aws_secretsmanager_secret.fleet_server_private_key.arn
}

output "fleet_s3_software_installers_config" {
  value = {
    bucket_name      = var.fleet_config.software_installers.create_bucket == true ? aws_s3_bucket.software_installers[0].bucket : var.fleet_config.software_installers.bucket_name
    s3_object_prefix = var.fleet_config.software_installers.s3_object_prefix
  }
}

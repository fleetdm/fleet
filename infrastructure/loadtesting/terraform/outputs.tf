output "private_subnets" {
  value = data.terraform_remote_state.shared.outputs.vpc.private_subnet_arns
}

output "fleet_migration_revision" {
  value = aws_ecs_task_definition.migration.revision
}

output "fleet_migration_subnets" {
  value = jsonencode(aws_ecs_service.fleet.network_configuration[0].subnets)
}

output "fleet_migration_security_groups" {
  value = jsonencode(aws_ecs_service.fleet.network_configuration[0].security_groups)
}

output "fleet_ecs_cluster_arn" {
  value = aws_ecs_cluster.fleet.arn
}

output "fleet_ecs_cluster_id" {
  value = aws_ecs_cluster.fleet.id
}

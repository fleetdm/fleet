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

output "fleet-backend-task-revision" {
  value = aws_ecs_task_definition.backend.revision
}

output "fleet-migration-task-revision" {
  value = aws_ecs_task_definition.migration.revision
}

output "redis_cluster_members" {
  value = toset(aws_elasticache_replication_group.default.member_clusters)
}

output "mysql_cluster_members" {
  value = toset(module.aurora_mysql.rds_cluster_instance_ids)
}

output "acm_certificate_arn" {
  value = aws_acm_certificate.dogfood_fleetdm_com.arn
}

output "load_balancer_arn_suffix" {
  value = aws_alb.main.arn_suffix
}

output "target_group_arn_suffix" {
  value = aws_alb_target_group.main.arn_suffix
}

output "fleet_min_capacity" {
  value = var.fleet_min_capacity
}

output "fleet_ecs_service_name" {
  value = aws_ecs_service.fleet.name
}

output "aws_alb_target_group_name" {
  value = aws_alb_target_group.main.name
}

output "aws_alb_name" {
  value = aws_alb.main.name
}
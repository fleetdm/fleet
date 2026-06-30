output "server_url" {
  value = "https://${aws_route53_record.main.fqdn}"
}

output "internal_alb_dns_name" {
  value = resource.aws_lb.internal.dns_name
}

output "internal_alb_listener_arn" {
  description = "Internal ALB HTTP listener ARN for adding host-based rules"
  value       = resource.aws_lb_listener.internal.arn
}

output "internal_alb_https_listener_arn" {
  description = "Internal ALB HTTPS listener ARN for adding host-based rules"
  value       = resource.aws_lb_listener.internal_https.arn
}

output "internal_alb_zone_id" {
  description = "Internal ALB hosted zone ID for Route53 alias records"
  value       = resource.aws_lb.internal.zone_id
}

output "internal_alb_security_group_id" {
  description = "Internal ALB security group ID"
  value       = resource.aws_security_group.internal.id
}

output "ecs_cluster" {
  sensitive = true
  value     = module.loadtest.byo-db.byo-ecs.service.cluster
}

output "security_groups" {
  sensitive = true
  value     = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].security_groups
}

output "ecs_arn" {
  sensitive = true
  value     = module.loadtest.byo-db.byo-ecs.iam_role_arn
}

output "ecs_execution_arn" {
  sensitive = true
  value     = module.loadtest.byo-db.byo-ecs.execution_iam_role_arn
}

output "logging_config" {
  sensitive = true
  value     = module.loadtest.byo-db.byo-ecs.logging_config
}

output "enroll_secret_arn" {
  sensitive = true
  value     = data.aws_secretsmanager_secret_version.enroll_secret.arn
}

output "vpc_subnets" {
  sensitive   = true
  value       = data.terraform_remote_state.shared.outputs.vpc.private_subnets
  description = "VPC private subnets from shared fleet-vpc"
}

output "rds_cluster_endpoint" {
  description = "RDS Aurora cluster writer endpoint"
  value       = module.loadtest.rds.cluster_endpoint
}

output "rds_cluster_reader_endpoint" {
  description = "RDS Aurora cluster reader endpoint"
  value       = module.loadtest.rds.cluster_reader_endpoint
}

output "rds_cluster_master_username" {
  description = "RDS Aurora cluster master username"
  value       = module.loadtest.rds.cluster_master_username
  sensitive   = true
}

output "rds_cluster_database_name" {
  description = "RDS Aurora cluster database name"
  value       = module.loadtest.rds.cluster_database_name
}

output "rds_security_group_id" {
  description = "Security group ID for the RDS cluster"
  value       = module.loadtest.rds.security_group_id
}

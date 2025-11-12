output "server_url" {
  value = "https://${aws_route53_record.main.fqdn}"
}

output "internal_alb_dns_name" {
  value = resource.aws_lb.internal.dns_name
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

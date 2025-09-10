output "server_url" {
  sensitive = true
  value     = "https://${aws_route53_record.main.fqdn}"
}

output "internal_alb_dns_name" {
  sensitive = true
  value     = var.run_migrations ? resource.aws_lb.internal.dns_name : ""
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

output "enroll_secret" {
  sensitive = true
  value     = data.aws_secretsmanager_secret_version.enroll_secret.secret_string
}

output "enroll_secret_arn" {
  sensitive = true
  value     = data.aws_secretsmanager_secret_version.enroll_secret.arn
}

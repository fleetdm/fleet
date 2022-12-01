module "ecs" {
  source = "./byo-db"
}

module "rds" {
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "7.6.0"

  name           = var.rds_config.name
  engine         = "aurora-postgresql"
  engine_version = var.rds_config.engine_version
  instance_class = var.rds_config.instance_class

  vpc_id  = var.vpc_id
  subnets = var.rds_config.subnets

  allowed_security_groups = concat(module.ecs.security_group, var.rds_config.allowed_security_groups)
  allowed_cidr_blocks     = var.rds_config.allowed_cidr_blocks

  storage_encrypted   = true
  apply_immediately   = var.rds_config.apply_immediately
  monitoring_interval = var.rds_config.monitoring_interval

  db_parameter_group_name         = var.rds_config.db_parameter_group_name == null ? aws_rds_paramater_group.main.id : var.rds_config.db_parameter_group_name
  db_cluster_parameter_group_name = var.rds_config.db_cluster_parameter_group_name == null ? aws_rds_cluster_paramater_group.main.id : var.rds_config.db_cluster_parameter_group_name

  enabled_cloudwatch_logs_exports = var.rds_config.enabled_cloudwatch_logs_exports
}

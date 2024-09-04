module "byo-db" {
  source = "./byo-db"
  vpc_id = var.vpc_config.vpc_id
  fleet_config = merge(var.fleet_config, {
    database = {
      address             = module.rds.cluster_endpoint
      rr_address          = module.rds.cluster_reader_endpoint
      database            = "fleet"
      user                = "fleet"
      password_secret_arn = module.secrets-manager-1.secret_arns["${var.rds_config.name}-database-password"]
    }
    redis = {
      address = "${module.redis.endpoint}:${module.redis.port}"
    }
    networking = {
      subnets         = var.vpc_config.networking.subnets
      security_groups = var.fleet_config.networking.security_groups
      ingress_sources = var.fleet_config.networking.ingress_sources
    }
  })
  ecs_cluster      = var.ecs_cluster
  migration_config = var.migration_config
  alb_config       = var.alb_config
}

resource "random_password" "rds" {
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

module "rds" {
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "7.6.0"

  name           = var.rds_config.name
  engine         = "aurora-mysql"
  engine_version = var.rds_config.engine_version
  instance_class = var.rds_config.instance_class

  instances = {
    one = {}
    two = {}
  }

  vpc_id  = var.vpc_config.vpc_id
  subnets = var.rds_config.subnets

  allowed_security_groups = concat(tolist(module.byo-db.byo-ecs.non_circular.security_groups), var.rds_config.allowed_security_groups)
  allowed_cidr_blocks     = var.rds_config.allowed_cidr_blocks

  performance_insights_enabled = true
  storage_encrypted            = true
  apply_immediately            = var.rds_config.apply_immediately
  monitoring_interval          = var.rds_config.monitoring_interval

  db_parameter_group_name         = var.rds_config.db_parameter_group_name == null ? aws_db_parameter_group.main[0].id : var.rds_config.db_parameter_group_name
  db_cluster_parameter_group_name = var.rds_config.db_cluster_parameter_group_name == null ? aws_rds_cluster_parameter_group.main[0].id : var.rds_config.db_cluster_parameter_group_name

  enabled_cloudwatch_logs_exports = var.rds_config.enabled_cloudwatch_logs_exports
  master_username                 = var.rds_config.master_username
  master_password                 = random_password.rds.result
  database_name                   = "fleet"
  skip_final_snapshot             = true
  snapshot_identifier             = var.rds_config.snapshot_identifier

  preferred_maintenance_window = var.rds_config.preferred_maintenance_window

  cluster_tags = var.rds_config.cluster_tags
}

data "aws_subnet" "redis" {
  for_each = toset(var.redis_config.subnets)
  id       = each.value
}

module "redis" {
  source  = "cloudposse/elasticache-redis/aws"
  version = "0.53.0"

  name                          = var.redis_config.name
  replication_group_id          = var.redis_config.replication_group_id == null ? var.redis_config.name : var.redis_config.replication_group_id
  elasticache_subnet_group_name = var.redis_config.elasticache_subnet_group_name == null ? var.redis_config.name : var.redis_config.elasticache_subnet_group_name
  availability_zones            = var.redis_config.availability_zones
  vpc_id                        = var.vpc_config.vpc_id
  description                   = "Fleet Redis"
  #allowed_security_group_ids = concat(var.redis_config.allowed_security_group_ids, module.byo-db.ecs.security_group)
  subnets                    = var.redis_config.subnets
  cluster_size               = var.redis_config.cluster_size
  instance_type              = var.redis_config.instance_type
  apply_immediately          = var.redis_config.apply_immediately
  automatic_failover_enabled = var.redis_config.automatic_failover_enabled
  engine_version             = var.redis_config.engine_version
  family                     = var.redis_config.family
  at_rest_encryption_enabled = var.redis_config.at_rest_encryption_enabled
  transit_encryption_enabled = var.redis_config.transit_encryption_enabled
  parameter                  = var.redis_config.parameter
  log_delivery_configuration = var.redis_config.log_delivery_configuration
  additional_security_group_rules = [{
    type        = "ingress"
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = var.redis_config.allowed_cidrs
  }]
  tags = var.redis_config.tags
}

module "secrets-manager-1" {
  source  = "lgallard/secrets-manager/aws"
  version = "0.6.1"

  secrets = {
    "${var.rds_config.name}-database-password" = {
      description             = "fleet-database-password"
      recovery_window_in_days = 0
      secret_string           = module.rds.cluster_master_password
    },
  }
}

resource "aws_db_parameter_group" "main" {
  count       = var.rds_config.db_parameter_group_name == null ? 1 : 0
  name        = var.rds_config.name
  family      = "aurora-mysql8.0"
  description = "fleet"

  dynamic "parameter" {
    for_each = var.rds_config.db_parameters
    content {
      name  = parameter.key
      value = parameter.value
    }
  }
}

resource "aws_rds_cluster_parameter_group" "main" {
  count       = var.rds_config.db_cluster_parameter_group_name == null ? 1 : 0
  name        = var.rds_config.name
  family      = "aurora-mysql8.0"
  description = "fleet"

  dynamic "parameter" {
    for_each = var.rds_config.db_cluster_parameters
    content {
      name  = parameter.key
      value = parameter.value
    }
  }

}

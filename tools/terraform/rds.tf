locals {
  name = "fleetdm"
}

resource "random_password" "database_password" {
  length  = 16
  special = false
}

resource "aws_secretsmanager_secret" "database_password_secret" {
  name = "/fleet/database/password/master"
}

resource "aws_secretsmanager_secret_version" "database_password_secret_version" {
  secret_id     = aws_secretsmanager_secret.database_password_secret.id
  secret_string = random_password.database_password.result
}

module "aurora_mysql" {
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "~> 3.0"

  name                   = "${local.name}-mysql"
  engine                 = "aurora-mysql"
  engine_mode            = "serverless"
  engine_version = ""
  storage_encrypted      = true
  username               = "fleet"
  password               = random_password.database_password.result
  create_random_password = false
  database_name          = "fleet"
  enable_http_endpoint   = true

  vpc_id                = module.vpc.vpc_id
  subnets               = module.vpc.database_subnets
  create_security_group = true
  allowed_cidr_blocks   = module.vpc.private_subnets_cidr_blocks

  replica_scale_enabled = false
  replica_count         = 0

  monitoring_interval = 60

  apply_immediately   = true
  skip_final_snapshot = true

  db_parameter_group_name         = aws_db_parameter_group.example_mysql.id
  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.example_mysql.id

  scaling_configuration = {
    auto_pause               = true
    min_capacity             = 2
    max_capacity             = 16
    seconds_until_auto_pause = 300
    timeout_action           = "ForceApplyCapacityChange"
  }
}

resource "aws_db_parameter_group" "example_mysql" {
  name        = "${local.name}-aurora-db-mysql-parameter-group"
  family      = "aurora-mysql5.7"
  description = "${local.name}-aurora-db-mysql-parameter-group"
}

resource "aws_rds_cluster_parameter_group" "example_mysql" {
  name        = "${local.name}-aurora-mysql-cluster-parameter-group"
  family      = "aurora-mysql5.7"
  description = "${local.name}-aurora-mysql-cluster-parameter-group"
}
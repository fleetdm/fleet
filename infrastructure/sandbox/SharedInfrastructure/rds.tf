resource "random_password" "database_password" {
  length  = 16
  special = false
}

resource "aws_kms_key" "main" {
  description             = "${var.prefix}-${random_pet.db_secret_postfix.id}"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "random_pet" "db_secret_postfix" {
  length = 1
}

resource "aws_secretsmanager_secret" "database_password_secret" {
  name       = "/fleet/database/password/master-2-${random_pet.db_secret_postfix.id}"
  kms_key_id = aws_kms_key.main.id
}

resource "aws_secretsmanager_secret_version" "database_password_secret_version" {
  secret_id     = aws_secretsmanager_secret.database_password_secret.id
  secret_string = random_password.database_password.result
}

resource "aws_secretsmanager_secret" "mysql" {
  name       = "/fleet/database/password/mysql-${random_pet.db_secret_postfix.id}"
  kms_key_id = aws_kms_key.main.id
}

output "mysql_secret" {
  value = aws_secretsmanager_secret.mysql
}

output "mysql_secret_kms" {
  value = aws_kms_key.main
}

resource "aws_secretsmanager_secret_version" "mysql" {
  secret_id = aws_secretsmanager_secret.mysql.id
  secret_string = jsonencode({
    endpoint = module.main.cluster_endpoint
    username = module.main.cluster_master_username
    password = module.main.cluster_master_password
  })
}

module "main" {
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "7.6.0"

  name           = var.prefix
  engine         = "aurora-mysql"
  engine_version = "5.7.mysql_aurora.2.11.3"
  engine_mode    = "serverless"

  storage_encrypted            = true
  master_username              = "fleet"
  master_password              = random_password.database_password.result
  create_random_password       = false
  enable_http_endpoint         = false
  performance_insights_enabled = true

  vpc_id                          = var.vpc.vpc_id
  subnets                         = var.vpc.database_subnets
  create_security_group           = true
  allowed_security_groups         = var.allowed_security_groups
  allowed_cidr_blocks             = ["10.0.0.0/8"]
  kms_key_id                      = aws_kms_key.main.arn
  performance_insights_kms_key_id = aws_kms_key.main.arn

  monitoring_interval = 60

  apply_immediately   = true
  skip_final_snapshot = true

  db_parameter_group_name         = aws_db_parameter_group.main.id
  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.main.id

  scaling_configuration = {
    auto_pause               = true
    min_capacity             = 32
    max_capacity             = 64
    seconds_until_auto_pause = 300
    timeout_action           = "ForceApplyCapacityChange"
  }
}

resource "aws_db_parameter_group" "main" {
  name        = "${var.prefix}-aurora-db-mysql-parameter-group"
  family      = "aurora-mysql5.7"
  description = "${var.prefix}-aurora-db-mysql-parameter-group"
}

resource "aws_rds_cluster_parameter_group" "main" {
  name        = "${var.prefix}-aurora-mysql-cluster-parameter-group"
  family      = "aurora-mysql5.7"
  description = "${var.prefix}-aurora-mysql-cluster-parameter-group"
}

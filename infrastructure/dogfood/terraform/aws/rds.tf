resource "random_password" "database_password" {
  length  = 32
  special = false
}
// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future.
resource "aws_secretsmanager_secret" "database_password_secret" { #tfsec:ignore:aws-ssm-secret-use-customer-key:exp:2022-07-01
  name                    = "/fleet/database/password/master"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "database_password_secret_version" {
  secret_id     = aws_secretsmanager_secret.database_password_secret.id
  secret_string = random_password.database_password.result
}

// if you want to use RDS Serverless option prefer the following commented block
//module "aurora_mysql_serverless" {
//  source  = "terraform-aws-modules/rds-aurora/aws"
//  version = "5.2.0"
//
//  name                   = "${local.name}-mysql"
//  engine                 = "aurora-mysql"
//  engine_mode            = "serverless"
//  storage_encrypted      = true
//  username               = "fleet"
//  password               = random_password.database_password.result
//  create_random_password = false
//  database_name          = "fleet"
//  enable_http_endpoint   = true
//
//  vpc_id                = module.vpc.vpc_id
//  subnets               = module.vpc.database_subnets
//  create_security_group = true
//  allowed_cidr_blocks   = concat(module.vpc.private_subnets_cidr_blocks, var.extra_security_group_cidrs)
//
//  replica_scale_enabled = false
//  replica_count         = 0
//
//  monitoring_interval = 60
//
//  apply_immediately   = true
//  skip_final_snapshot = true
//
//  db_parameter_group_name         = aws_db_parameter_group.example_mysql.id
//  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.example_mysql.id
//
//  scaling_configuration = {
//    auto_pause               = true
//    min_capacity             = 2
//    max_capacity             = 16
//    seconds_until_auto_pause = 300
//    timeout_action           = "ForceApplyCapacityChange"
//  }
//}

variable "db_instance_type_writer" {
  default = "db.t4g.medium"
}
variable "db_instance_type_reader" {
  default = "db.t4g.medium"
}

module "aurora_mysql" {
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "5.2.0"

  name                  = "${local.name}-mysql-iam"
  engine                = "aurora-mysql"
  engine_version        = "8.0.mysql_aurora.3.05.2"
  instance_type         = var.db_instance_type_writer
  instance_type_replica = var.db_instance_type_reader

  iam_database_authentication_enabled = true
  storage_encrypted                   = true
  username                            = var.database_user
  password                            = random_password.database_password.result
  create_random_password              = false
  database_name                       = var.database_name
  enable_http_endpoint                = false
  backup_retention_period             = var.rds_backup_retention_period
  snapshot_identifier                 = var.rds_initial_snapshot
  #performance_insights_enabled       = true

  vpc_id                = module.vpc.vpc_id
  subnets               = module.vpc.database_subnets
  create_security_group = true
  allowed_cidr_blocks   = concat(module.vpc.private_subnets_cidr_blocks, var.extra_security_group_cidrs)

  replica_count         = 1
  replica_scale_enabled = true
  replica_scale_min     = 1
  replica_scale_max     = 3

  monitoring_interval           = 60
  iam_role_name                 = "${local.name}-rds-enhanced-monitoring"
  iam_role_use_name_prefix      = true
  iam_role_description          = "${local.name} RDS enhanced monitoring IAM role"
  iam_role_path                 = "/autoscaling/"
  iam_role_max_session_duration = 7200

  apply_immediately   = true
  skip_final_snapshot = true

  db_parameter_group_name         = aws_db_parameter_group.example_mysql.id
  db_cluster_parameter_group_name = aws_rds_cluster_parameter_group.example_mysql.id
}

resource "aws_db_parameter_group" "example_mysql" {
  name        = "${local.name}-aurora-db-mysql-parameter-group"
  family      = "aurora-mysql8.0"
  description = "${local.name}-aurora-db-mysql-parameter-group"
}

resource "aws_rds_cluster_parameter_group" "example_mysql" {
  name        = "${local.name}-aurora-mysql-cluster-parameter-group"
  family      = "aurora-mysql8.0"
  description = "${local.name}-aurora-mysql-cluster-parameter-group"
}

resource "null_resource" "rds_guardian" {
  triggers = {
    rds_cluster = module.aurora_mysql.rds_cluster_endpoint
  }

  lifecycle {
    prevent_destroy = true
  }
}

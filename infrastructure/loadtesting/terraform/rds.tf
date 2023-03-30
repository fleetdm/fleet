resource "random_password" "database_password" {
  length  = 16
  special = false
}

resource "random_pet" "db_secret_postfix" {
  length = 1
}

resource "aws_secretsmanager_secret" "database_password_secret" {
  name                    = "/fleet/database/password/master-2-${random_pet.db_secret_postfix.id}"
  kms_key_id              = aws_kms_key.main.id
  # No need to keep these around to potentially break re-using the same
  # workspace.
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "database_password_secret_version" {
  secret_id     = aws_secretsmanager_secret.database_password_secret.id
  secret_string = random_password.database_password.result
}

module "aurora_mysql" { #tfsec:ignore:aws-rds-enable-performance-insights-encryption tfsec:ignore:aws-rds-encrypt-cluster-storage-data tfsec:ignore:aws-vpc-add-description-to-security-group
  source  = "terraform-aws-modules/rds-aurora/aws"
  version = "5.3.0"

  name                  = "${local.name}-mysql"
  engine                = "aurora-mysql"
  engine_version        = "5.7.mysql_aurora.2.10.3"
  instance_type         = var.db_instance_type
  instance_type_replica = var.db_instance_type

  iam_database_authentication_enabled = true
  storage_encrypted                   = true
  username                            = "fleet"
  password                            = random_password.database_password.result
  create_random_password              = false
  database_name                       = "fleet"
  enable_http_endpoint                = false
  performance_insights_enabled        = true
  enabled_cloudwatch_logs_exports     = ["slowquery"]

  vpc_id                  = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  vpc_security_group_ids  = [aws_security_group.backend.id]
  subnets                 = data.terraform_remote_state.shared.outputs.vpc.database_subnets
  create_security_group   = true
  allowed_cidr_blocks     = concat(data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks, local.vpn_cidr_blocks)
  # Old Jump box?
  # allowed_security_groups = ["sg-0063a978193fdf7ee"]

  replica_count         = 2
  snapshot_identifier   = "arn:aws:rds:us-east-2:917007347864:cluster-snapshot:cleaned"

  monitoring_interval           = 60
  iam_role_name                 = "${local.name}-rds"
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
  family      = "aurora-mysql5.7"
  description = "${local.name}-aurora-db-mysql-parameter-group"
}

resource "aws_rds_cluster_parameter_group" "example_mysql" {
  name        = "${local.name}-aurora-mysql-cluster-parameter-group"
  family      = "aurora-mysql5.7"
  description = "${local.name}-aurora-mysql-cluster-parameter-group"
}

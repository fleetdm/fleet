resource "random_password" "database_password" {
  length  = 16
  special = false
}

resource "random_pet" "db_secret_postfix" {
  length = 1
}

resource "aws_secretsmanager_secret" "database_password_secret" {
  name       = "/fleet/database/password/master-2-${random_pet.db_secret_postfix.id}"
  kms_key_id = aws_kms_key.main.id
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
  version = "7.7.1"

  name           = "${local.name}-mysql"
  engine         = "aurora-mysql"
  engine_version = "8.0.mysql_aurora.3.05.2"
  instance_class = var.db_instance_type

  instances = {
    one = {}
#    two = {}
  }

  iam_database_authentication_enabled = true
  storage_encrypted                   = true
  master_username                     = "fleet"
  master_password                     = random_password.database_password.result
  create_random_password              = false
  database_name                       = "fleet"
  enable_http_endpoint                = false
  performance_insights_enabled        = true
  enabled_cloudwatch_logs_exports     = ["slowquery"]

  vpc_id                 = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  vpc_security_group_ids = [aws_security_group.backend.id]
  subnets                = data.terraform_remote_state.shared.outputs.vpc.database_subnets
  create_security_group  = true
  allowed_cidr_blocks    = concat(data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks, local.vpn_cidr_blocks)
  # Old Jump box?
  # allowed_security_groups = ["sg-0063a978193fdf7ee"]
  create_db_cluster_parameter_group = true
  db_cluster_parameter_group_family = "aurora-mysql8.0"
  db_cluster_parameter_group_name   = "${local.name}-mysql-parameters"
  db_cluster_parameter_group_parameters = [
    {
      name         = "innodb_print_all_deadlocks"
      value        = "1"
      apply_method = "immediate"
    }
  ]

  snapshot_identifier = "arn:aws:rds:us-east-2:917007347864:cluster-snapshot:cleaned-8-0"

  monitoring_interval           = 60
  iam_role_name                 = "${local.name}-rds"
  iam_role_use_name_prefix      = true
  iam_role_description          = "${local.name} RDS enhanced monitoring IAM role"
  iam_role_path                 = "/autoscaling/"
  iam_role_max_session_duration = 7200

  apply_immediately   = true
  skip_final_snapshot = true
}

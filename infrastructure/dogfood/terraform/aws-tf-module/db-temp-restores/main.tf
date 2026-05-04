terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "dogfood/db-temp-restores/terraform.tfstate"
    workspace_key_prefix = "dogfood"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-dogfood"
    }
  }
}

provider "aws" {
  default_tags {
    tags = {
      environment = "dogfood"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws-tf-module/db-temp-restores"
      workspace   = terraform.workspace
    }
  }
}

# Read shared VPC from remote state
data "terraform_remote_state" "dogfood" {
  backend   = "s3"
  workspace = "fleet"
  config = {
    bucket         = "fleet-terraform-remote-state"
    key            = "fleet"
    region         = "us-east-2"
    dynamodb_table = "fleet-terraform-state-lock"
  }
}

data "aws_region" "current" {}

locals {
  tags = {
    environment           = "dogfood"
    terraform             = "https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws-tf-module/db-temp-restores"
  }
  allowed_subnets = [
    "10.255.1.0/24",
    "10.255.2.0/24",
    "10.255.3.0/24",
  ]
  developers = [
    "tim",
    "jordan",
    "lucas",
    "magnus",
    "robert",
    "jorge",
    "ian",
    "victor"
  ]
  customers = {
    fleet-dogfood-1 = "arn:aws:rds:us-east-2:160035666661:cluster-snapshot:rds:fleet-dogfood-1-2026-03-18-02-06"
  }
}

resource "aws_security_group" "db_restore_vpn_access" {
  name        = "db-restore-vpn-access"
  description = "Access restored db via VPN"
  vpc_id      = data.terraform_remote_state.dogfood.outputs.vpc.vpc_id
  tags = {
    Name = "db-restore-vpn-access"
  }
}

# Ingress: allow TCP 3306 from each VPN subnet
resource "aws_vpc_security_group_ingress_rule" "mysql_from_vpn_subnets" {
  for_each = toset(local.allowed_subnets)

  security_group_id = aws_security_group.db_restore_vpn_access.id
  cidr_ipv4         = each.value
  ip_protocol       = "tcp"
  from_port         = 3306
  to_port           = 3306
  description       = "Allow MySQL from ${each.value}"
}

# Egress: allow ALL traffic
resource "aws_vpc_security_group_egress_rule" "all_egress" {
  security_group_id = aws_security_group.db_restore_vpn_access.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
  description       = "Allow all egress"
}

resource "aws_db_subnet_group" "shared" {
  name        = "db-restore-shared"
  description = "Shared subnet group for Aurora VPN Restores"
  subnet_ids  = data.terraform_remote_state.dogfood.outputs.vpc.database_subnets

  tags = {
    Name    = "db-restore-shared"
    Purpose = "Shared Aurora DB subnet group for VPN restores"
  }
}


module "dev_restore_dbs" {
  for_each = local.customers
  source   = "terraform-aws-modules/rds-aurora/aws"
  version  = "~> 9.0"

  name                   = "${each.key}-restore"
  engine                 = "aurora-mysql"
  db_subnet_group_name   = aws_db_subnet_group.shared.name
  vpc_security_group_ids = [aws_security_group.db_restore_vpn_access.id]

  snapshot_identifier = each.value

  # Ensure the master password is managed by a Secrets Manager secret
  manage_master_user_password = true

  # Provide/confirm the master user (must match/override snapshot’s username)
  master_username = "fleet" # <-- set to your cluster admin username

  # Single instance, fixed size
  instances = {
    db1 = {
      instance_class = "db.serverless"
    }
  }

  serverlessv2_scaling_configuration = {
    min_capacity = 0 # or 0.5 if you prefer a warm floor
    max_capacity = 4 # keep modest to cap cost; raise if needed
  }

  # Use existing parameter groups if provided
  db_cluster_parameter_group_name = each.key
  db_parameter_group_name         = each.key

  # --- KMS encryption at rest (use default AWS managed key) ---
  storage_encrypted = true
  # kms_key_id      = null  # Omit to use the default AWS-managed key for RDS

  create_security_group = false

  apply_immediately   = true
  deletion_protection = false
  skip_final_snapshot = true

  tags = {
    Name                  = "${each.key}-db-restore"
    Customer              = each.key
    Purpose               = "DB restore with VPN access"
  }
}

data "aws_secretsmanager_secret" "aurora_master" {
  for_each = local.customers
  arn      = module.dev_restore_dbs[each.key].cluster_master_user_secret[0].secret_arn
}

data "aws_secretsmanager_secret_version" "aurora_master" {
  for_each  = local.customers
  secret_id = data.aws_secretsmanager_secret.aurora_master[each.key].id
}

module "mysql_dev_access" {
  for_each = local.customers
  source   = "./mysql_dev_access"

  # Inline endpoint (host:port)
  endpoint = "${module.dev_restore_dbs[each.key].cluster_endpoint}:3306"

  # Inline extraction of username/password from the secret.
  # Works for both JSON-formatted secrets ({"username":"...","password":"..."})
  # and plain-string secrets (fallback).
  admin_username = try(
    jsondecode(data.aws_secretsmanager_secret_version.aurora_master[each.key].secret_string).username,
    "fleet" # fallback if username not present
  )

  admin_password = try(
    jsondecode(data.aws_secretsmanager_secret_version.aurora_master[each.key].secret_string).password,
    data.aws_secretsmanager_secret_version.aurora_master[each.key].secret_string
  )

  developers    = local.developers
  database_name = "fleet"

  # Ensure the cluster exists before we try to connect
  depends_on = [module.dev_restore_dbs]
}

output "databases" {
  value     = module.dev_restore_dbs
  sensitive = true
}

output "developer_passwords" {
  value     = module.mysql_dev_access
  sensitive = true
}

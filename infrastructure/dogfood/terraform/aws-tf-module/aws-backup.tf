provider "aws" {
  region = "us-west-2"
  alias  = "replica"
}

#####################
#### PERMISSIONS ####
#####################
data "aws_iam_policy_document" "aws_backup_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["backup.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "aws_backup" {
  name               = "aws_backup_role"
  assume_role_policy = data.aws_iam_policy_document.aws_backup_assume_role.json
}

resource "aws_iam_role_policy_attachment" "aws_backup_policy" {
  role       = resource.aws_iam_role.aws_backup.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

##############
### AURORA ###
##############

###
## Source Key and backup vault
###
resource "aws_kms_key" "aws_backup_aurora_source" {
  description = "Source CMEK for Aurora - AWS Backups"
}

resource "aws_backup_vault" "aws_backup_aurora_source" {
  name        = "backup_aurora_vault_source"
  kms_key_arn = resource.aws_kms_key.aws_backup_aurora_source.arn
}

###
## Destination Key and backup vault
###
resource "aws_kms_key" "aws_backup_aurora_destination" {
  provider    = aws.replica
  description = "Destination CMEK for Aurora - AWS Backups"
}

resource "aws_backup_vault" "aws_backup_aurora_destination" {
  provider    = aws.replica
  name        = "backup_aurora_vault_destination"
  kms_key_arn = resource.aws_kms_key.aws_backup_aurora_destination.arn
}

###
## Starts snapshot copy within 1 hour of scheduled plan start time
## Completes backup within 2 hours of start time
###
resource "aws_backup_plan" "snapshot_backup_plan" {
  name = "aurora_snapshot_backup_plan"
  rule {
    rule_name         = "daily_aurora_snapshot_backup"
    target_vault_name = resource.aws_backup_vault.aws_backup_aurora_source.name
    schedule          = "cron(0 5 * * ? *)"
    start_window      = 60
    completion_window = 120

    lifecycle {
      delete_after = 7
    }

    copy_action {
      destination_vault_arn = resource.aws_backup_vault.aws_backup_aurora_destination.arn
    }
  }
}

###
## Backups will occur on:
## Aurora Cluster Snapshots that are tagged with Backup = true
###
resource "aws_backup_selection" "snapshot_selection" {
  name         = "aurora_snapshot_backup_selection"
  iam_role_arn = resource.aws_iam_role.aws_backup.arn
  plan_id      = resource.aws_backup_plan.snapshot_backup_plan.id
  resources = [
    "arn:aws:rds:us-east-2:160035666661:cluster:*"
  ]
  condition {
    string_equals {
      key   = "aws:ResourceTag/backup"
      value = "true"
    }
  }
}

##############
##### S3 #####
##############

###
## Source Key and backup vault
###
resource "aws_kms_key" "aws_backup_s3_source" {
  description = "Source CMEK for s3 - AWS Backups"
}

resource "aws_backup_vault" "aws_backup_s3_source" {
  name        = "backup_s3_vault_source"
  kms_key_arn = resource.aws_kms_key.aws_backup_s3_source.arn
}

###
## Destination Key and backup vault
###
resource "aws_kms_key" "aws_backup_s3_destination" {
  provider    = aws.replica
  description = "Destination CMEK for s3 - AWS Backups"
}

resource "aws_backup_vault" "aws_backup_s3_destination" {
  provider    = aws.replica
  name        = "backup_s3_vault_destination"
  kms_key_arn = resource.aws_kms_key.aws_backup_s3_destination.arn
}

###
## Starts snapshot copy within 1 hour of scheduled plan start time
## Completes backup within 2 hours of start time
###
resource "aws_backup_plan" "s3_backup_plan" {
  name = "s3_backup_plan"
  rule {
    rule_name         = "daily_s3_backup"
    target_vault_name = resource.aws_backup_vault.aws_backup_s3_source.name
    schedule          = "cron(0 5 * * ? *)"
    start_window      = 60
    completion_window = 120

    lifecycle {
      delete_after = 7
    }

    copy_action {
      destination_vault_arn = resource.aws_backup_vault.aws_backup_s3_destination.arn
    }
  }
}

###
## Backups will occur on:
## S3 bucks that are tagged with backup = true
###
resource "aws_backup_selection" "s3_selection" {
  name         = "s3_backup_selection"
  iam_role_arn = resource.aws_iam_role.aws_backup.arn
  plan_id      = resource.aws_backup_plan.s3_backup_plan.id
  resources = [
    "arn:aws:s3:::*-software-installers-*"
  ]
  condition {
    string_equals {
      key   = "aws:ResourceTag/backup"
      value = "true"
    }
  }
}

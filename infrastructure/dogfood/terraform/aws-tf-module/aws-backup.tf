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

data "aws_iam_policy_document" "aws_backup_policy" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:DescribeTable",
      "dynamodb:CreateBackup"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:DescribeBackup",
      "dynamodb:DeleteBackup"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*/backup/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:AddTagsToResource",
      "rds:ListTagsForResource",
      "rds:DescribeDBSnapshots",
      "rds:CreateDBSnapshot",
      "rds:CopyDBSnapshot",
      "rds:DescribeDBInstances",
      "rds:CreateDBClusterSnapshot",
      "rds:DescribeDBClusters",
      "rds:DescribeDBClusterSnapshots",
      "rds:CopyDBClusterSnapshot",
      "rds:DescribeDBClusterAutomatedBackups"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:DeleteDBInstanceAutomatedBackup"
    ]
    resources = ["arn:aws:rds:*:*:auto-backup:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:ModifyDBCluster"
    ]
    resources = ["arn:aws:rds:*:*:cluster:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:DeleteDBClusterAutomatedBackup"
    ]
    resources = ["arn:aws:rds:*:*:cluster-auto-backup:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:ModifyDBInstance"
    ]
    resources = ["arn:aws:rds:*:*:db:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:DeleteDBSnapshot",
      "rds:ModifyDBSnapshotAttribute"
    ]
    resources = ["arn:aws:rds:*:*:snapshot:awsbackup:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:DeleteDBClusterSnapshot",
      "rds:ModifyDBClusterSnapshotAttribute"
    ]
    resources = ["arn:aws:rds:*:*:cluster-snapshot:awsbackup:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey"
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:StringLike"
      variable = "kms:ViaService"
      values = [
        "dynamodb.*.amazonaws.com",
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:DescribeKey"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:CreateGrant",
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:Bool"
      variable = "kms:GrantIsForAWSResource"
      values = [
        "true"
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "tag:GetResources"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:StartAwsBackupJob",
      "dynamodb:ListTagsOfResource"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "backup:TagResource"
    ]
    resources = ["arn:aws:backup:*:*:recovery-point:*"]
    condition {
      test     = "ForAnyValue:StringEquals"
      variable = "aws:PrincipalAccount"
      values = [
        "&{aws:ResourceAccount}"
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "cloudwatch:GetMetricData"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "events:DeleteRule",
      "events:PutTargets",
      "events:DescribeRule",
      "events:EnableRule",
      "events:PutRule",
      "events:RemoveTargets",
      "events:ListTargetsByRule",
      "events:DisableRule"
    ]
    resources = ["arn:aws:events:*:*:rule/AwsBackupManagedRule*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "events:ListRules"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:DescribeKey"
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:StringLike"
      variable = "kms:ViaService"
      values = [
        "s3.*.amazonaws.com",
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:GetBucketTagging",
      "s3:GetInventoryConfiguration",
      "s3:ListBucketVersions",
      "s3:ListBucket",
      "s3:GetBucketVersioning",
      "s3:GetBucketLocation",
      "s3:GetBucketAcl",
      "s3:PutInventoryConfiguration",
      "s3:GetBucketNotification",
      "s3:PutBucketNotification"
    ]
    resources = ["arn:aws:s3:::*software-installers*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:GetObjectAcl",
      "s3:GetObject",
      "s3:GetObjectVersionTagging",
      "s3:GetObjectVersionAcl",
      "s3:GetObjectTagging",
      "s3:GetObjectVersion"
    ]
    resources = ["arn:aws:s3:::*software-installers*/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:ListAllMyBuckets"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "backup:TagResource"
    ]
    resources = ["arn:aws:backup:*:*:recovery-point:*"]
    condition {
      test     = "ForAnyValue:StringEquals"
      variable = "aws:PrincipalAccount"
      values = [
        "&{aws:ResourceAccount}"
      ]
    }
  }
}

data "aws_iam_policy_document" "aws_restore_policy" {
  statement {
    effect = "Allow"
    actions = [
      "dynamodb:Scan",
      "dynamodb:Query",
      "dynamodb:UpdateItem",
      "dynamodb:PutItem",
      "dynamodb:GetItem",
      "dynamodb:DeleteItem",
      "dynamodb:BatchWriteItem",
      "dynamodb:DescribeTable"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:RestoreTableFromBackup"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*/backup/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:DescribeDBInstances",
      "rds:DescribeDBSnapshots",
      "rds:ListTagsForResource",
      "rds:RestoreDBInstanceFromDBSnapshot",
      "rds:DeleteDBInstance",
      "rds:AddTagsToResource",
      "rds:DescribeDBClusters",
      "rds:RestoreDBClusterFromSnapshot",
      "rds:DeleteDBCluster",
      "rds:RestoreDBInstanceToPointInTime",
      "rds:DescribeDBClusterSnapshots",
      "rds:RestoreDBClusterToPointInTime",
      "rds:CreateTenantDatabase",
      "rds:DeleteTenantDatabase"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:DescribeKey"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:Encrypt",
      "kms:GenerateDataKey",
      "kms:ReEncryptTo",
      "kms:ReEncryptFrom",
      "kms:GenerateDataKeyWithoutPlaintext"
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:StringLike"
      variable = "kms:ViaService"
      values = [
        "dynamodb.*.amazonaws.com",
        "rds.*.amazonaws.com"
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:CreateGrant"
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:Bool"
      variable = "kms:GrantIsForAWSResource"
      values = [
        "true"
      ]
    }
  }

  statement {
    effect = "Allow"
    actions = [
      "rds:CreateDBInstance"
    ]
    resources = ["arn:aws:rds:*:*:db:*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:RestoreTableFromAwsBackup"
    ]
    resources = ["arn:aws:dynamodb:*:*:table/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:CreateBucket",
      "s3:ListBucketVersions",
      "s3:ListBucket",
      "s3:GetBucketVersioning",
      "s3:GetBucketLocation",
      "s3:PutBucketVersioning",
      "s3:PutBucketOwnershipControls",
      "s3:GetBucketOwnershipControls"
    ]
    resources = ["arn:aws:s3:::*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "s3:GetObject",
      "s3:GetObjectVersion",
      "s3:DeleteObject",
      "s3:PutObjectVersionAcl",
      "s3:GetObjectVersionAcl",
      "s3:GetObjectTagging",
      "s3:PutObjectTagging",
      "s3:GetObjectAcl",
      "s3:PutObjectAcl",
      "s3:ListMultipartUploadParts",
      "s3:PutObject"
    ]
    resources = ["arn:aws:s3:::*/*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:DescribeKey",
      "kms:GenerateDataKey",
      "kms:Decrypt"
    ]
    resources = ["*"]
    condition {
      test     = "ForAnyValue:StringLike"
      variable = "kms:ViaService"
      values   = ["s3.*.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "aws_backup" {
  name               = "aws_backup_role"
  assume_role_policy = data.aws_iam_policy_document.aws_backup_assume_role.json

  inline_policy {
    name   = "aws-backup-restore-policy"
    policy = data.aws_iam_policy_document.aws_backup_policy.json
  }

  inline_policy {
    name   = "aws-backup-backup-policy"
    policy = data.aws_iam_policy_document.aws_restore_policy.json
  }
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
## S3 buckets that are tagged with backup = true
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

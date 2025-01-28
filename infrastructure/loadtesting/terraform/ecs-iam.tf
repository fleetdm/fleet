data "aws_iam_policy_document" "fleet" {
  statement {
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.database_password_secret.arn, data.aws_secretsmanager_secret.license.arn, aws_secretsmanager_secret.fleet_server_private_key.arn]
  }

  // useful when there is a static number of mysql cluster members
  # dynamic "statement" {
  #   for_each = module.aurora_mysql.rds_cluster_instance_dbi_resource_ids
  #   content {
  #     effect    = "Allow"
  #     actions   = ["rds-db:connect"]
  #     resources = ["arn:aws:rds-db:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:dbuser:${statement.value}/${module.aurora_mysql.cluster_master_username}"]
  #   }
  # }

  // allow access to any database via IAM that has the var.database_user user
  // useful when you are autoscaling mysql read replicas dynamically
  statement {
    effect    = "Allow"
    actions   = ["rds-db:connect"]
    resources = ["arn:aws:rds-db:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:dbuser:*/${module.aurora_mysql.cluster_master_username}"]
  }

  statement {
    effect = "Allow"
    actions = [
      "firehose:DescribeDeliveryStream",
      "firehose:PutRecord",
      "firehose:PutRecordBatch",
    ]
    resources = [aws_kinesis_firehose_delivery_stream.osquery_results.arn, aws_kinesis_firehose_delivery_stream.osquery_status.arn]
  }

  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [aws_kms_key.main.arn, data.terraform_remote_state.shared.outputs.ecr-kms.arn]
  }
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["ecs.amazonaws.com", "ecs-tasks.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "main" {
  name               = "${local.prefix}-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "role_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.main.name
}

resource "aws_iam_policy" "main" {
  name   = "${local.prefix}-iam-policy"
  policy = data.aws_iam_policy_document.fleet.json
}

resource "aws_iam_role_policy_attachment" "attachment" {
  policy_arn = aws_iam_policy.main.arn
  role       = aws_iam_role.main.name
}

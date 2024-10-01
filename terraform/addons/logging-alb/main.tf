data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

locals {

  kms_policies = concat([{
    actions = ["kms:*"],
    principals = [{
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }]
    resources = ["*"]

    },
    {
      actions = [
        "kms:Encrypt*",
        "kms:Decrypt*",
        "kms:ReEncrypt*",
        "kms:GenerateDataKey*",
        "kms:Describe*",
      ]
      resources = ["*"]
      principals = [{
        type        = "Service"
        identifiers = ["logs.${data.aws_region.current.name}.amazonaws.com"]
      }]
  }], var.extra_kms_policies)

}


data "aws_iam_policy_document" "kms" {
  dynamic "statement" {
    for_each = local.kms_policies
    content {
      sid       = try(statement.value.sid, "")
      actions   = try(statement.value.actions, [])
      resources = try(statement.value.resources, [])
      effect    = try(statement.value.effect, null)
      dynamic "principals" {
        for_each = try(statement.value.principals, [])
        content {
          type        = principals.value.type
          identifiers = principals.value.identifiers
        }
      }
      dynamic "condition" {
        for_each = try(statement.value.conditions, [])
        content {
          test     = condition.value.test
          variable = condition.value.variable
          values   = condition.value.values
        }
      }
    }
  }
}

data "aws_iam_policy_document" "s3_log_bucket" {
  count = var.extra_s3_log_policies == [] ? 0 : 1
  dynamic "statement" {
    for_each = var.extra_s3_log_policies
    content {
      sid       = try(statement.value.sid, "")
      actions   = try(statement.value.actions, [])
      resources = try(statement.value.resources, [])
      effect    = try(statement.value.effect, null)
      dynamic "principals" {
        for_each = try(statement.value.principals, [])
        content {
          type        = principals.value.type
          identifiers = principals.value.identifiers
        }
      }
      dynamic "condition" {
        for_each = try(statement.value.conditions, [])
        content {
          test     = condition.value.test
          variable = condition.value.variable
          values   = condition.value.values
        }
      }
    }
  }
}

data "aws_iam_policy_document" "s3_athena_bucket" {
  count = var.extra_s3_athena_policies == [] ? 0 : 1
  dynamic "statement" {
    for_each = var.extra_s3_athena_policies
    content {
      sid       = try(statement.value.sid, "")
      actions   = try(statement.value.actions, [])
      resources = try(statement.value.resources, [])
      effect    = try(statement.value.effect, null)
      dynamic "principals" {
        for_each = try(statement.value.principals, [])
        content {
          type        = principals.value.type
          identifiers = principals.value.identifiers
        }
      }
      dynamic "condition" {
        for_each = try(statement.value.conditions, [])
        content {
          test     = condition.value.test
          variable = condition.value.variable
          values   = condition.value.values
        }
      }
    }
  }
}

resource "aws_kms_key" "logs" {
  policy              = data.aws_iam_policy_document.kms.json
  enable_key_rotation = true
}

resource "aws_kms_alias" "logs_alias" {
  name_prefix   = "alias/${var.prefix}-logs"
  target_key_id = aws_kms_key.logs.id
}

module "s3_bucket_for_logs" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.15.1"

  bucket = "${var.prefix}-alb-logs"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  attach_policy                         = var.extra_s3_log_policies != []
  policy                                = var.extra_s3_log_policies != [] ? data.aws_iam_policy_document.s3_log_bucket[0].json : null
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  server_side_encryption_configuration = {
    rule = {
      bucket_key_enabled = true
      apply_server_side_encryption_by_default = {
        sse_algorithm = "AES256"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = var.s3_transition_days
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days                         = var.s3_expiration_days
        # Always resets to false anyhow showing terraform changes constantly
        expired_object_delete_marker = false
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = var.s3_newer_noncurrent_versions
        days                      = var.s3_noncurrent_version_expiration_days
      }
    }
  ]
}

resource "aws_athena_database" "logs" {
  count  = var.enable_athena == true ? 1 : 0
  name   = replace("${var.prefix}-alb-logs", "-", "_")
  bucket = module.athena-s3-bucket[0].s3_bucket_id
}

module "athena-s3-bucket" {
  count   = var.enable_athena == true ? 1 : 0
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "3.15.1"

  bucket = "${var.prefix}-alb-logs-athena"

  # Allow deletion of non-empty bucket
  force_destroy = true

  attach_elb_log_delivery_policy        = true # Required for ALB logs
  attach_lb_log_delivery_policy         = true # Required for ALB/NLB logs
  attach_deny_insecure_transport_policy = true
  attach_require_latest_tls_policy      = true
  attach_policy                         = var.extra_s3_athena_policies != []
  policy                                = var.extra_s3_athena_policies != [] ? data.aws_iam_policy_document.s3_athena_bucket[0].json : null
  block_public_acls                     = true
  block_public_policy                   = true
  ignore_public_acls                    = true
  restrict_public_buckets               = true
  server_side_encryption_configuration = {
    rule = {
      apply_server_side_encryption_by_default = {
        kms_master_key_id = aws_kms_key.logs.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
  lifecycle_rule = [
    {
      id      = "log"
      enabled = true

      transition = [
        {
          days          = var.s3_transition_days
          storage_class = "ONEZONE_IA"
        }
      ]
      expiration = {
        days                         = var.s3_expiration_days
        # Always resets to false anyhow showing terraform changes constantly
        expired_object_delete_marker = false
      }
      noncurrent_version_expiration = {
        newer_noncurrent_versions = var.s3_newer_noncurrent_versions
        days                      = var.s3_noncurrent_version_expiration_days
      }
    }
  ]
}

resource "aws_athena_workgroup" "logs" {
  count = var.enable_athena == true ? 1 : 0
  name  = "${var.prefix}-logs"

  configuration {
    enforce_workgroup_configuration    = true
    publish_cloudwatch_metrics_enabled = true

    result_configuration {
      output_location = "s3://${module.athena-s3-bucket[0].s3_bucket_id}/output/"

      encryption_configuration {
        encryption_option = "SSE_KMS"
        kms_key_arn       = aws_kms_key.logs.arn
      }
    }
  }

  force_destroy = true
}

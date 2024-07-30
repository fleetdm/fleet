data "aws_iam_policy_document" "osquery_firehose_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["firehose.amazonaws.com"]
      type        = "Service"
    }
  }
}

data "aws_iam_policy_document" "firehose_policy" {
  statement {
    effect = "Allow"
    actions = [
      "s3:AbortMultipartUpload",
      "s3:GetBucketLocation",
      "s3:GetObject",
      "s3:ListBucket",
      "s3:ListBucketMultipartUploads",
      "s3:PutObject",
      "s3:PutObjectAcl" // required according to https://docs.aws.amazon.com/firehose/latest/dev/controlling-access.html#using-iam-s3
    ]
    resources = [
      aws_s3_bucket.destination.arn,
      "${aws_s3_bucket.destination.arn}/*",
    ]
  }

  statement {
    effect  = "Allow"
    actions = ["logs:PutLogEvents"]
    resources = [
      for name in keys(var.log_destinations) : "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:/aws/kinesisfirehose/${var.log_destinations[name].name}:*"
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey"
    ]
    resources = [data.aws_kms_alias.s3.arn]
  }

}

resource "aws_iam_role" "firehose" {
  assume_role_policy = data.aws_iam_policy_document.osquery_firehose_assume_role.json
}

resource "aws_iam_policy" "firehose" {
  policy = data.aws_iam_policy_document.firehose_policy.json
}

resource "aws_iam_role_policy_attachment" "firehose" {
  policy_arn = aws_iam_policy.firehose.arn
  role       = aws_iam_role.firehose.name
}

resource "aws_kms_key" "firehose_key" {
  count       = var.server_side_encryption_enabled && length(var.kms_key_arn) == 0 ? 1 : 0
  description = "KMS key for encrypting Firehose data."
}

resource "aws_kinesis_firehose_delivery_stream" "fleet_log_destinations" {
  for_each    = var.log_destinations
  name        = each.value.name
  destination = "extended_s3"

  dynamic "server_side_encryption" {
    for_each = var.server_side_encryption_enabled ? [1] : []
    content {
      enabled  = var.server_side_encryption_enabled
      key_arn  = length(var.kms_key_arn) > 0 ? var.kms_key_arn : aws_kms_key.firehose_key[0].arn
      key_type = "CUSTOMER_MANAGED_CMK"
    }
  }

  extended_s3_configuration {
    bucket_arn          = aws_s3_bucket.destination.arn
    role_arn            = aws_iam_role.firehose.arn
    prefix              = each.value.prefix
    error_output_prefix = each.value.error_output_prefix
    buffering_size      = each.value.buffering_size
    buffering_interval  = each.value.buffering_interval
    compression_format  = each.value.compression_format
  }
}

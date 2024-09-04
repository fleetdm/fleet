resource "aws_iam_role" "fleet_role" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = [var.fleet_iam_role_arn]
      type        = "AWS"
    }
    dynamic "condition" {
      for_each = length(var.sts_external_id) > 0 ? [1] : []
      content {
        test     = "StringEquals"
        variable = "sts:ExternalId"
        values   = [var.sts_external_id]
      }
    }
  }
}

data "aws_iam_policy_document" "firehose" {
  statement {
    effect = "Allow"
    actions = [
      "firehose:DescribeDeliveryStream",
      "firehose:PutRecord",
      "firehose:PutRecordBatch",
    ]
    resources = [
      for stream in aws_kinesis_firehose_delivery_stream.fleet_log_destinations : stream.arn
    ]
  }

  dynamic "statement" {
    for_each = var.server_side_encryption_enabled ? [1] : []

    content {
      effect = "Allow"
      actions = [
        "kms:Decrypt",
        "kms:GenerateDataKey",
      ]
      resources = [
        length(var.kms_key_arn) > 0 ? var.kms_key_arn : aws_kms_key.firehose_key[0].arn
      ]
    }
  }

}

resource "aws_iam_policy" "fleet_firehose" {
  policy = data.aws_iam_policy_document.firehose.json
}

resource "aws_iam_role_policy_attachment" "fleet_firehose" {
  policy_arn = aws_iam_policy.fleet_firehose.arn
  role       = aws_iam_role.fleet_role.name
}
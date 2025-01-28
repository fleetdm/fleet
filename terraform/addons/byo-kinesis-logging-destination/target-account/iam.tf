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

data "aws_iam_policy_document" "kinesis" {
  statement {
    effect = "Allow"
    actions = [
      "kinesis:DescribeStreamSummary",
      "kinesis:DescribeStream",
      "kinesis:PutRecord",
      "kinesis:PutRecords"
    ]
    resources = [
      for stream in aws_kinesis_stream.fleet_log_destination : stream.arn
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey",
    ]
    resources = [
      aws_kms_key.kinesis_key.arn
    ]
  }
}

resource "aws_iam_policy" "fleet_kinesis" {
  policy = data.aws_iam_policy_document.kinesis.json
}

resource "aws_iam_role_policy_attachment" "fleet_kinesis" {
  policy_arn = aws_iam_policy.fleet_kinesis.arn
  role       = aws_iam_role.fleet_role.name
}
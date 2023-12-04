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
      aws_kinesis_firehose_delivery_stream.osquery_results.arn,
      aws_kinesis_firehose_delivery_stream.osquery_status.arn,
      aws_kinesis_firehose_delivery_stream.fleet_audit.arn
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "kms:Decrypt",
      "kms:GenerateDataKey"
    ]
    resources = [aws_kms_key.firehose.arn]
  }

}

resource "aws_iam_policy" "fleet_firehose" {
  policy = data.aws_iam_policy_document.firehose.json
}

resource "aws_iam_role_policy_attachment" "fleet_firehose" {
  policy_arn = aws_iam_policy.fleet_firehose.arn
  role       = aws_iam_role.fleet_role.name
}
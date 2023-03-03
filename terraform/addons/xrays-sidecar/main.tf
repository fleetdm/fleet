data "aws_region" "current" {}

data "aws_iam_policy_document" "main" {
  statement {
    actions = [
      "xray:PutTraceSegments",
      "xray:PutTelemetryRecords",
      "xray:GetSamplingRules",
      "xray:GetSamplingTargets",
      "xray:GetSamplingStatisticSummaries",
    ]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "main" {
  policy = data.aws_iam_policy_document.main.json
}

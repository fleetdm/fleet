data "aws_iam_policy_document" "fleet-assume-role" {
  statement {
    effect    = "Allow"
    actions   = ["sts:AssumeRole"]
    resources = [var.iam_role_arn]
  }
}

resource "aws_iam_policy" "fleet-assume-role" {
  policy = data.aws_iam_policy_document.fleet-assume-role.json
}
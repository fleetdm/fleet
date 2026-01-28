data "aws_iam_policy_document" "license" {
  statement {
    effect  = "Allow"
    actions = ["secretsmanager:GetSecretValue"]
    resources = [
      data.aws_secretsmanager_secret.license.arn
    ]
  }
}

resource "aws_iam_policy" "license" {
  name   = "${local.customer}-license-iam-policy"
  policy = data.aws_iam_policy_document.license.json
}

data "aws_iam_policy_document" "enroll" {
  statement {
    effect  = "Allow"
    actions = ["secretsmanager:GetSecretValue"]
    resources = [
      data.aws_secretsmanager_secret_version.enroll_secret.arn
    ]
  }
}

resource "aws_iam_policy" "enroll" {
  name        = "${local.customer}-enroll-policy"
  description = "IAM policy that Fleet application uses to define access to AWS resources"
  policy      = data.aws_iam_policy_document.enroll.json
}

resource "aws_iam_role_policy_attachment" "enroll" {
  policy_arn = aws_iam_policy.enroll.arn
  role       = "${local.customer}-execution-role"

  depends_on = [
    module.loadtest
  ]
}
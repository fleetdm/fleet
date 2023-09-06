data "aws_region" "current" {}

resource "aws_secretsmanager_secret" "apn" {
  name = var.apn_secret_name
}

resource "aws_secretsmanager_secret" "scep" {
  name = var.scep_secret_name
}

resource "aws_secretsmanager_secret" "dep" {
  count = var.dep_secret_name == null ? 0 : 1
  name  = var.dep_secret_name
}

data "aws_iam_policy_document" "main" {
  statement {
    actions = ["secretsmanager:GetSecretValue"]
    resources = concat([
      aws_secretsmanager_secret.apn.arn,
      aws_secretsmanager_secret.scep.arn,
    ], var.dep_secret_name == null ? [] : [aws_secretsmanager_secret.dep[0].arn])
  }
}

resource "aws_iam_policy" "main" {
  policy = data.aws_iam_policy_document.main.json
}

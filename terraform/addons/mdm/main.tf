data "aws_region" "current" {}

resource "aws_secretsmanager_secret" "apn" {
  name = var.apn_secret_name
}

resource "aws_secretsmanager_secret" "scep" {
  name = var.scep_secret_name
}

resource "aws_secretsmanager_secret" "dep" {
  name = var.dep_secret_name
}

data "aws_iam_policy_document" "main" {
  statement {
    actions = ["secretsmanager:GetSecretValue"]
    resources = [
      aws_secretsmanager_secret.apn.arn,
      aws_secretsmanager_secret.scep.arn,
      aws_secretsmanager_secret.dep.arn,
    ]
  }
}

resource "aws_iam_policy" "main" {
  policy = data.aws_iam_policy_document.main.json
}

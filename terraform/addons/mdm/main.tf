data "aws_region" "current" {}

resource "aws_secretsmanager_secret" "apn" {
  count = var.apn_secret_name == null ? 0 : 1
  name  = var.apn_secret_name

  recovery_window_in_days = "0"
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_secretsmanager_secret" "scep" {
  name = var.scep_secret_name

  recovery_window_in_days = "0"
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_secretsmanager_secret" "abm" {
  count = var.abm_secret_name == null ? 0 : 1
  name  = var.abm_secret_name

  recovery_window_in_days = "0"
  lifecycle {
    create_before_destroy = true
  }
}

data "aws_iam_policy_document" "main" {
  statement {
    actions = ["secretsmanager:GetSecretValue"]
    resources = concat(var.enable_apple_mdm == false ? [] : [aws_secretsmanager_secret.apn[0].arn],
      [aws_secretsmanager_secret.scep.arn],
    var.abm_secret_name == null ? [] : [aws_secretsmanager_secret.abm[0].arn])
  }
}

resource "aws_iam_policy" "main" {
  policy = data.aws_iam_policy_document.main.json
}

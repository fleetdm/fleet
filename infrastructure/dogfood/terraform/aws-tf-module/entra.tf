variable "entra_api_key" {}

resource "aws_iam_policy" "entra_conditional_access" {
  name   = "fleet-entra-conditional-access"
  policy = data.aws_iam_policy_document.entra_conditional_access.json
}

data "aws_iam_policy_document" "entra_conditional_access" {
  statement {
    actions = [
      "secretsmanager:GetSecretValue",
    ]
    resources = [aws_secretsmanager_secret.entra_conditional_access.arn]
  }
}

resource "aws_secretsmanager_secret" "entra_conditional_access" {
  name = "dogfood-entra-conditional-access"
}

resource "aws_secretsmanager_secret_version" "entra_api_key" {
  secret_id     = aws_secretsmanager_secret.entra_conditional_access.id
  secret_string = base64encode(var.entra_api_key)
}

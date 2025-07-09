resource "aws_secretsmanager_secret" "entra_conditional_access" {
  name = "dogfood-entra-conditional-access"
}

resource "aws_secretsmanager_secret_version" "entra_api_key" {
  secret_id     = aws_secretsmanager_secret.entra_conditional_access.id
  secret_string = base64encode(var.entra_api_key)
}

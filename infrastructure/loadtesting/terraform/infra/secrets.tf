data "aws_secretsmanager_secret" "license" {
  name = "/fleet/license"
}

data "aws_secretsmanager_secret_version" "enroll_secret" {
  secret_id = data.terraform_remote_state.shared.outputs.enroll_secret.id
}
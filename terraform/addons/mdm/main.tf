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

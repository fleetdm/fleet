resource "aws_secretsmanager_secret" "apn" {
  name = "apn"
}

resource "aws_secretsmanager_secret" "scep" {
  name = "scep"
}

resource "aws_secretsmanager_secret" "dep" {
  name = "dep"
}

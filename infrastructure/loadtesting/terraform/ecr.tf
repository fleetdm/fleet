resource "aws_kms_key" "main" {
  description             = "${local.prefix}-${random_pet.db_secret_postfix.id}"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "prometheus-to-cloudwatch" {
  name                 = "prometheus-to-cloudwatch"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }
}

resource "aws_ecr_repository" "fleet" {
  name                 = "fleet"
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }
}

data "aws_ecr_authorization_token" "token" {}

resource "docker_registry_image" "fleet" {
  name          = "${aws_ecr_repository.fleet.repository_url}:${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  keep_remotely = true

  build {
    context = "${path.cwd}/docker/"
    build_args = {
      TAG = var.tag
    }
    pull_parent = true
  }
}

data "docker_registry_image" "dockerhub" {
  name = "fleetdm/fleet:${var.tag}"
}

resource "docker_registry_image" "loadtest" {
  name          = "${aws_ecr_repository.fleet.repository_url}:loadtest-${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  keep_remotely = true

  build {
    context    = "${path.cwd}/docker/"
    dockerfile = "loadtest.Dockerfile"
    build_args = {
      TAG = var.tag
    }
    pull_parent = true
  }
}

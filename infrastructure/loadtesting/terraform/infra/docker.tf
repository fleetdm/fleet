data "aws_ecr_authorization_token" "token" {}

resource "random_pet" "db_secret_postfix" {
  length = 1
}

resource "aws_kms_key" "main" {
  description             = "${local.customer}-${random_pet.db_secret_postfix.id}"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "fleet" {
  name                 = local.customer
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }

  force_delete = true
}

resource "docker_image" "dockerhub" {
  name = "fleetdm/fleet:${var.tag}"
}

data "docker_registry_image" "dockerhub" {
  name = "fleetdm/fleet:${var.tag}"
}

resource "docker_tag" "fleet" {
  source_image = docker_image.dockerhub.name
  target_image = "${aws_ecr_repository.fleet.repository_url}:${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
}

resource "docker_registry_image" "fleet" {
  name          = docker_tag.fleet.target_image
  keep_remotely = true
}
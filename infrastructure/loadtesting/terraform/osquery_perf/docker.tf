data "aws_ecr_authorization_token" "token" {}

data "aws_ecr_repository" "fleet" {
  name = local.customer
}

resource "random_pet" "rand_image_key" {
  length = 1
}

resource "aws_kms_key" "main" {
  description             = "${local.customer}-osq-${random_pet.rand_image_key.id}"
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_ecr_repository" "loadtest" {
  name = "${local.customer}-osq"

  image_scanning_configuration {
    scan_on_push = true
  }

  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = aws_kms_key.main.arn
  }

  force_delete = true
}

resource "docker_registry_image" "loadtest" {
  name          = docker_image.loadtest.name
  keep_remotely = true
}

resource "docker_image" "loadtest" {
  name         = "${resource.aws_ecr_repository.loadtest.repository_url}:loadtest-${local.loadtest_tag}"
  keep_locally = true
  force_remove = true
  build {
    context    = "../docker/"
    dockerfile = "loadtest.Dockerfile"
    platform   = "linux/amd64"
    build_args = {
      TAG = local.loadtest_tag
    }
    pull_parent = true
  }
}

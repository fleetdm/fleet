resource "aws_ecr_repository" "prometheus-to-cloudwatch" {
  name                 = "prometheus-to-cloudwatch"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_repository" "fleet" {
  name                 = "fleet"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

data "aws_ecr_authorization_token" "token" {}

resource "docker_registry_image" "fleet" {
  name = "${aws_ecr_repository.fleet.repository_url}:${var.tag}"

  build {
    context = "${path.cwd}/docker/"
    build_args = {
      TAG = var.tag
    }
    pull_parent = true
  }
}

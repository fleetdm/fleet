data "aws_ecr_authorization_token" "token" {}

data "docker_registry_image" "dockerhub" {
  name = "fleetdm/fleet:${var.tag}"
}

data "aws_ecr_repository" "fleet" {
  name = local.customer
}

resource "docker_registry_image" "loadtest" {
  name          = docker_image.loadtest.name
  keep_remotely = true
}

resource "docker_image" "loadtest" {
  name         = "${data.aws_ecr_repository.fleet.repository_url}:loadtest-${local.loadtest_tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  keep_locally = true
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

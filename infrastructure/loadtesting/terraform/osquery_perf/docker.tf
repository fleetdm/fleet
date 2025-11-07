data "aws_ecr_authorization_token" "token" {}

data "aws_ecr_repository" "fleet" {
  name = local.customer
}

resource "docker_registry_image" "loadtest" {
  name          = docker_image.loadtest.name
  keep_remotely = true
}

resource "docker_image" "loadtest" {
  name         = "${data.aws_ecr_repository.fleet.repository_url}:loadtest-${local.loadtest_tag}"
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

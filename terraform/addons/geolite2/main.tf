terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.0.2"
    }
  }
}

# Write it to disk for usage
resource "local_file" "dockerfile" {
  filename = "${path.module}/Dockerfile"
  content  = templatefile(
    "${path.module}/Dockerfile.tpl",
    {
      fleet_image = var.fleet_image
    }
  )
}

# Build the new image
resource "docker_image" "maxmind_fleet" {
  name = var.destination_image

  build {
    context  = path.module
    platform = "linux/amd64"
    build_args = {
      LICENSE_KEY = var.license_key
    }
    pull_parent = true
  }
}

# push it to the specified repo
resource "docker_registry_image" "maxmind_fleet" {
  triggers = {
    fleet_digest = docker_image.maxmind_fleet.repo_digest
  }
  name          = docker_image.maxmind_fleet.name
  keep_remotely = true
}

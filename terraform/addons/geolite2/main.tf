terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.0.2"
    }
  }
}

# Build the new image
resource "docker_image" "maxmind_fleet" {
  name = var.destination_image

  build {
    context  = path.module
    platform = "linux/amd64"
    build_args = {
      FLEET_IMAGE = var.fleet_image
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

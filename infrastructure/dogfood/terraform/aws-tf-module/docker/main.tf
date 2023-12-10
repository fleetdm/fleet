terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.0.2"
    }
  }
}

variable "osquery_tag" {
  description = "The osquery tag to take from dockerhub to your ecr repo."
  type        = string
}

variable "ecr_repo" {
  description = "The ecr repo to push to"
  type        = string
}

resource "docker_image" "dockerhub" {
  name = "osquery/osquery:${var.osquery_tag}"
}

resource "docker_tag" "osquery" {
  source_image = docker_image.dockerhub.name
  # We can't include the sha256 when pushing even if they match
  target_image = "${var.ecr_repo}:${split("@sha256", var.osquery_tag)[0]}"
}

resource "docker_registry_image" "osquery" {
  name          = docker_tag.osquery.target_image
  keep_remotely = true
}

output "ecr_image" {
  value = docker_tag.osquery.target_image
}

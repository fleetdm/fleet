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

variable "osquery_version" {
  description = "The osquery version to push to your ecr repo."
  type        = string
}

variable "osquery_tags" {
  description = "The tags that you wish to push among the built images"
  type        = list(string)
}

variable "ecr_repo" {
  description = "The ecr repo to push to"
  type        = string
}

resource "local_file" "osquery_patch" {
  content         = templatefile("${path.module}/osquery-docker.patch.tmpl", { osquery_version = var.osquery_version })
  filename        = "${path.module}/osquery-docker.patch"
  file_permission = "0644"
}

resource "null_resource" "build_osquery" {
  depends_on = [local_file.osquery_patch]
  triggers = {
    osquery_version_changed = var.osquery_version
    osquery_tags_changed    = sha256(jsonencode(var.osquery_tags))
  }
  provisioner "local-exec" {
    working_dir = "${path.module}"
    command     = <<-EOT
      mkdir -p osquery
      cd osquery
      if [ "$(git remote -vvv | head -n1 | awk '{ print $2 }')" = "https://github.com/osquery/osquery.git" ]; then
        git reset --hard
        git pull
      else
        git clone https://github.com/osquery/osquery.git .
      fi
      git apply ../osquery-docker.patch
      cd tools/docker
      ./build.sh
    EOT
  }
}

resource "docker_tag" "osquery" {
  depends_on   = [null_resource.build_osquery]
  for_each     = toset(var.osquery_tags)
  source_image = "osquery/osquery:${each.key}"
  # We can't include the sha256 when pushing even if they match
  target_image = "${var.ecr_repo}:${each.key}"
}

resource "docker_registry_image" "osquery" {
  for_each      = toset(var.osquery_tags)
  name          = docker_tag.osquery[each.key].target_image
  keep_remotely = true
}

output "ecr_images" {
  value = { for docker_tag in docker_tag.osquery : split(":", docker_tag.target_image)[1] => docker_tag.target_image }
}

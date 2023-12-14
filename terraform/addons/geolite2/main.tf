data "template_file" "dockerfile" {
  template = "${file("${path.module}/Dockerfile.tmpl")}"
  vars = {
    fleet_image = vars.fleet_image
  }
}

resource "local_file" "dockerfile" {
  filename = "${path.module}/Dockerfile"
  content  = data.template_file.dockerfile.rendered
}

resource "docker_image" "maxmind_fleet" {
  name = var.destination_image

  build {
    context  = "${path.module}"
    platform = "linux/amd64"
    build_args = {
      LICENSE_KEY = var.license_key
    }
    pull_parent = true
  }
} 

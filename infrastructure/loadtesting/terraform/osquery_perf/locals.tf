locals {
  customer            = "fleet-${terraform.workspace}"
  loadtest_containers = var.loadtest_containers

  fleet_image  = var.tag
  loadtest_tag = var.git_branch != null ? var.git_branch : var.tag
}
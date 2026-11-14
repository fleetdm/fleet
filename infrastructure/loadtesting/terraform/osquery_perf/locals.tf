locals {
  customer            = "fleet-${terraform.workspace}"
  loadtest_containers = var.loadtest_containers

  # fleet_image  = var.tag
  loadtest_tag = var.git_tag_branch
  # Docker tags can't contain "/", which branch names often do.
  image_tag = replace(var.git_tag_branch, "/", "-")

  server_url         = "http://${data.terraform_remote_state.infra.outputs.internal_alb_dns_name}"
  osquery_perf_image = "${resource.aws_ecr_repository.loadtest.repository_url}:loadtest-${local.image_tag}"

  # Container index => its osquery-perf --seed (index + 1).
  stable_containers = var.stable_host_identities ? {
    for i in range(local.loadtest_containers) : tostring(i) => tostring(i + 1)
  } : {}
}

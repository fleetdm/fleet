
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "git_repository" "tf" {
  path = "${path.module}/../../../../"
}

module "osquery_perf" {
  source                     = "github.com/fleetdm/fleet-terraform//addons/osquery-perf?ref=tf-mod-addon-osquery-perf-v1.1.1"
  customer_prefix            = local.customer
  ecs_cluster                = data.terraform_remote_state.infra.outputs.ecs_cluster
  loadtest_containers        = local.loadtest_containers
  subnets                    = data.terraform_remote_state.shared.outputs.vpc.private_subnets
  security_groups            = data.terraform_remote_state.infra.outputs.security_groups
  ecs_iam_role_arn           = data.terraform_remote_state.infra.outputs.ecs_arn
  ecs_execution_iam_role_arn = data.terraform_remote_state.infra.outputs.ecs_execution_arn
  server_url                 = data.terraform_remote_state.infra.outputs.server_url
  osquery_perf_image         = "${data.aws_ecr_repository.fleet.repository_url}:loadtest-${local.loadtest_tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  extra_flags                = var.extra_flags
  logging_options            = data.terraform_remote_state.infra.outputs.logging_config
  enroll_secret              = data.terraform_remote_state.infra.outputs.enroll_secret
}



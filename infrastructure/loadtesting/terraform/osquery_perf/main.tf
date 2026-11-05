data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

data "git_repository" "tf" {
  directory = "${path.module}/../../../../"
}

module "osquery_perf" {
  count                      = var.stable_host_identities ? 0 : 1
  source                     = "github.com/fleetdm/fleet-terraform//addons/osquery-perf?ref=tf-mod-addon-osquery-perf-v1.2.2"
  customer_prefix            = local.customer
  ecs_cluster                = data.terraform_remote_state.infra.outputs.ecs_cluster
  loadtest_containers        = local.loadtest_containers
  subnets                    = data.terraform_remote_state.infra.outputs.vpc_subnets
  security_groups            = data.terraform_remote_state.infra.outputs.security_groups
  ecs_iam_role_arn           = data.terraform_remote_state.infra.outputs.ecs_arn
  ecs_execution_iam_role_arn = data.terraform_remote_state.infra.outputs.ecs_execution_arn
  server_url                 = local.server_url
  osquery_perf_image         = local.osquery_perf_image
  extra_flags                = var.extra_flags
  logging_options            = data.terraform_remote_state.infra.outputs.logging_config
  enroll_secret_arn          = data.terraform_remote_state.infra.outputs.enroll_secret_arn
  task_size                  = var.task_size
}

moved {
  from = module.osquery_perf
  to   = module.osquery_perf[0]
}

# With stable host identities, each container is its own service started with its own
# --seed, so a replaced or restarted task comes back as the same hosts. The module above
# runs every container as one service whose tasks are identical and don't know their index.
# The command mirrors the module's.
data "aws_secretsmanager_secret_version" "enroll_secret" {
  count     = var.stable_host_identities ? 1 : 0
  secret_id = data.terraform_remote_state.infra.outputs.enroll_secret_arn
}

resource "aws_ecs_task_definition" "osquery_perf_stable" {
  for_each                 = local.stable_containers
  family                   = "${local.customer}-osquery-perf-${each.key}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = data.terraform_remote_state.infra.outputs.ecs_execution_arn
  task_role_arn            = data.terraform_remote_state.infra.outputs.ecs_arn
  cpu                      = var.task_size.cpu
  memory                   = var.task_size.memory
  container_definitions = jsonencode([
    {
      name        = "osquery-perf"
      image       = local.osquery_perf_image
      cpu         = var.task_size.cpu
      memory      = var.task_size.memory
      mountPoints = []
      volumesFrom = []
      essential   = true
      ulimits = [
        {
          softLimit = 999999,
          hardLimit = 999999,
          name      = "nofile"
        }
      ]
      networkMode = "awsvpc"
      logConfiguration = {
        logDriver = "awslogs"
        options   = data.terraform_remote_state.infra.outputs.logging_config
      }
      workingDirectory = "/go",
      # --seed goes last so a --seed in extra_flags can't give every container the same hosts.
      command = concat([
        "/go/osquery-perf",
        "-enroll_secret", data.aws_secretsmanager_secret_version.enroll_secret[0].secret_string,
        "-host_count", "500",
        "-server_url", local.server_url,
        "--policy_pass_prob", "0.5",
        "--start_period", "5m",
      ], var.extra_flags, ["--seed", each.value])
    }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_ecs_service" "osquery_perf_stable" {
  for_each        = local.stable_containers
  name            = "osquery_perf_${each.key}"
  launch_type     = "FARGATE"
  cluster         = data.terraform_remote_state.infra.outputs.ecs_cluster
  task_definition = aws_ecs_task_definition.osquery_perf_stable[each.key].arn
  desired_count   = 1
  # Stop the old task before starting its replacement: two tasks with the same seed would
  # run the same hosts at once and fight over their node keys.
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100

  network_configuration {
    subnets         = data.terraform_remote_state.infra.outputs.vpc_subnets
    security_groups = data.terraform_remote_state.infra.outputs.security_groups
  }
}

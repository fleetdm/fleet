resource "aws_ecs_service" "loadtest" {
  name                               = "loadtest"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.fleet.id
  task_definition                    = aws_ecs_task_definition.loadtest.arn
  desired_count                      = var.loadtest_containers
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  network_configuration {
    subnets         = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }
}

resource "aws_ecs_task_definition" "loadtest" {
  family                   = "${local.prefix}-loadtest"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode(
    [
      {
        name        = "loadtest"
        image       = docker_registry_image.loadtest.name
        cpu         = 256
        memory      = 512
        mountPoints = []
        volumesFrom = []
        essential   = true
        ulimits = [
          {
            softLimit = 9999,
            hardLimit = 9999,
            name      = "nofile"
          }
        ]
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.backend.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "loadtest"
          }
        },
        workingDirectory = "/go",
        command = [
          "/go/osquery-perf",
          "-enroll_secret", data.aws_secretsmanager_secret_version.enroll_secret.secret_string,
          "-host_count", "250",
          "-server_url", "http://${aws_lb.internal.dns_name}",
          "--policy_pass_prob", "0.5",
          "--start_period", "5m",
          "--orbit_prob", "1",
          "--mdm_prob", "1",
          "--mdm_scep_challenge", "foo",
          "--os_templates", "macos_14.1.2:250",
          "--mdm_check_in_interval", "1m"
        ]
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

data "aws_secretsmanager_secret_version" "enroll_secret" {
  secret_id = data.terraform_remote_state.shared.outputs.enroll_secret.id
}

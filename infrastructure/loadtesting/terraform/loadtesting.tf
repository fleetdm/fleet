locals {
  hosts_per_container = 500
  # Stable string keys for for_each; numeric values preserved for math
  loadtest_instances = { for i in range(var.loadtest_containers) : tostring(i) => i }
}

# ----------------------------
# ECS Task Definitions
# ----------------------------
resource "aws_ecs_task_definition" "loadtest" {
  for_each = local.loadtest_instances

  family                   = "loadtest-${local.prefix}-${each.key}"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 512
  memory                   = 1024

  container_definitions = jsonencode([
    {
      name        = "loadtest"
      image       = docker_registry_image.loadtest.name
      cpu         = 512
      memory      = 1024
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
          awslogs-stream-prefix = "loadtest-${each.key}"
        }
      }
      workingDirectory = "/go"
      command = [
        "/go/osquery-perf",
        "-enroll_secret", data.aws_secretsmanager_secret_version.enroll_secret.secret_string,
        "-host_count", tostring(local.hosts_per_container),
        # If you would like to run distributed mode, uncomment these two lines
        # "-total_host_count", tostring(var.loadtest_containers * local.hosts_per_container),
        # "-host_index_offset", tostring(each.value * local.hosts_per_container),
        "-server_url", "http://${aws_lb.internal.dns_name}",
        "--policy_pass_prob", "0.5",
        "--start_period", "5m",
        "--orbit_prob", "0.0"
      ]
    }
  ])

  lifecycle {
    create_before_destroy = true
  }
}

# ----------------------------
# ECS Services
# ----------------------------
resource "aws_ecs_service" "loadtest" {
  for_each = local.loadtest_instances

  name                               = "loadtest-${each.key}"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.fleet.id
  task_definition                    = aws_ecs_task_definition.loadtest[each.key].arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  network_configuration {
    subnets         = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }
}

# ----------------------------
# Secrets
# ----------------------------
data "aws_secretsmanager_secret_version" "enroll_secret" {
  secret_id = data.terraform_remote_state.shared.outputs.enroll_secret.id
}


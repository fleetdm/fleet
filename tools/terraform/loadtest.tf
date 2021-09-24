resource "aws_ecs_cluster" "osquery-perf" {
  #  count = var.loadtesting ? 1 : 0

  name = "${var.prefix}-osquery-perf"
}

resource "aws_ecs_service" "osquery-perf" {
  #  count = var.loadtesting ? 1 : 0

  name                               = "osquery-perf"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.osquery-perf.id
  task_definition                    = aws_ecs_task_definition.osquery-perf.arn
  desired_count                      = var.osquery_host_count

  network_configuration {
    subnets         = module.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }

  depends_on = [aws_alb_listener.http, aws_alb_listener.https-fleetdm]
}


resource "aws_ecs_task_definition" "osquery-perf" {
  #  count = var.loadtesting ? 1 : 0

  family                   = "osquery-perf"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 512
  memory                   = 4096
  container_definitions = jsonencode(
  [
    {
      name        = "osquery-perf"
      image       = "917007347864.dkr.ecr.us-east-2.amazonaws.com/osquery-perf"
      cpu         = 512
      memory      = 4096
      mountPoints = []
      volumesFrom = []
      essential   = true
      networkMode = "awsvpc"
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.osquery-perf.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "osquery-perf"
        }
      }
    }
  ])
}

resource "aws_appautoscaling_target" "osquery_ecs_target" {
  #  count = var.loadtesting ? 1 : 0

  max_capacity       = var.osquery_host_count
  min_capacity       = var.osquery_host_count
  resource_id        = "service/${aws_ecs_cluster.osquery-perf.name}/${aws_ecs_service.osquery-perf.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_cloudwatch_log_group" "osquery-perf" {
  #  count = var.loadtesting ? 1 : 0

  name              = "osquery-perf"
  retention_in_days = 1
}

data "aws_region" "current" {}

resource "aws_ecs_service" "fleet" {
  name                               = "fleet"
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.backend.arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 30

  load_balancer {
    target_group_arn = var.fleet_config.loadbalancer.arn
    container_name   = "fleet"
    container_port   = 8080
  }

  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = var.fleet_config.networking.subnets
    security_groups = var.fleet_config.networking.security_groups
  }
}

resource "aws_ecs_task_definition" "backend" {
  family                   = "fleet"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = var.fleet_config.iam_role_arn == null ? aws_iam_role.main[0].arn : var.fleet_config.iam_role_arn
  execution_role_arn       = aws_iam_role.execution.arn
  cpu                      = var.fleet_config.cpu
  memory                   = var.fleet_config.mem
  container_definitions = jsonencode(
    [
      {
        name        = "fleet"
        image       = var.fleet_config.image
        cpu         = var.fleet_config.cpu
        memory      = var.fleet_config.mem
        mountPoints = []
        volumesFrom = []
        essential   = true
        portMappings = [
          {
            # This port is the same that the contained application also uses
            containerPort = 8080
            protocol      = "tcp"
          }
        ]
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = var.fleet_config.awslogs.name == null ? aws_cloudwatch_log_group.main[0].name : var.fleet_config.awslogs.name
            awslogs-region        = var.fleet_config.awslogs.name == null ? data.aws_region.current.name : var.fleet_config.awslogs.region
            awslogs-stream-prefix = var.fleet_config.awslogs.prefix
          }
        },
        ulimits = [
          {
            name      = "nofile"
            softLimit = 999999
            hardLimit = 999999
          }
        ],
        secrets = [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = var.fleet_config.database.password_secret_arn
          },
          {
            name      = "FLEET_MYSQL_READ_REPLICA_PASSWORD"
            valueFrom = var.fleet_config.database.password_secret_arn
          }
        ]
        environment = [
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = var.fleet_config.database.user
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = var.fleet_config.database.database
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = var.fleet_config.database.address
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_USERNAME"
            value = var.fleet_config.database.user
            }, {
            name  = "FLEET_MYSQL_READ_REPLICA_DATABASE"
            value = var.fleet_config.database.database
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_ADDRESS"
            value = var.fleet_config.database.rr_address == null ? var.fleet_config.database.address : var.fleet_config.database.rr_address
          },
          {
            name  = "FLEET_REDIS_ADDRESS"
            value = var.fleet_config.redis.address
          },
          {
            name  = "FLEET_REDIS_USE_TLS"
            value = tostring(var.fleet_config.redis.use_tls)
          },
          {
            name  = "FLEET_OSQUERY_STATUS_LOG_PLUGIN"
            value = "firehose"
          },
          {
            name  = "FLEET_OSQUERY_RESULT_LOG_PLUGIN"
            value = "firehose"
          },
          {
            name  = "FLEET_SERVER_TLS"
            value = "false"
          },
        ]
      }
  ])
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = var.fleet_config.autoscaling.max_capacity
  min_capacity       = var.fleet_config.autoscaling.min_capacity
  resource_id        = "service/${var.ecs_cluster}/${aws_ecs_service.fleet.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_policy_memory" {
  name               = "fleet-memory-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
    target_value = var.fleet_config.autoscaling.memory_tracking_target_value
  }
}

resource "aws_appautoscaling_policy" "ecs_policy_cpu" {
  name               = "fleet-cpu-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs_target.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs_target.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs_target.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }

    target_value = var.fleet_config.autoscaling.cpu_tracking_target_value
  }
}

resource "aws_cloudwatch_log_group" "main" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key:exp:2022-07-01
  count             = var.fleet_config.awslogs.name == null ? 1 : 0
  name              = "fleetdm"
  retention_in_days = var.fleet_config.awslogs.retention
}

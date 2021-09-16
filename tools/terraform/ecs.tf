//resource "aws_route53_record" "record" {
//  name = "fleetdm"
//  type = "A"
//  zone_id = "Z046188311R47QSK245X"
//  alias {
//    evaluate_target_health = false
//    name = aws_alb.main.dns_name
//    zone_id = aws_alb.main.zone_id
//  }
//}

resource "aws_alb" "main" {
  name            = "fleetdm"
  internal        = false
  security_groups = [aws_security_group.lb.id, aws_security_group.backend.id]
  subnets         = module.vpc.public_subnets
}

resource "aws_alb_target_group" "main" {
  name                 = "fleetdm"
  protocol             = "HTTP"
  target_type          = "ip"
  port                 = "8080"
  vpc_id               = module.vpc.vpc_id
  deregistration_delay = 30

  load_balancing_algorithm_type = "least_outstanding_requests"

  health_check {
    path                = "/healthz"
    matcher             = "200"
    timeout             = 10
    interval            = 15
    healthy_threshold   = 5
    unhealthy_threshold = 5
  }

  depends_on = [aws_alb.main]
}

resource "aws_alb_listener" "main" {
  load_balancer_arn = aws_alb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate_validation.dogfood_fleetctl_com.certificate_arn

  default_action {
    target_group_arn = aws_alb_target_group.main.arn
    type             = "forward"
  }
}

resource "aws_alb_listener" "http" {
  load_balancer_arn = aws_alb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

resource "aws_ecs_cluster" "fleet" {
  name = "${var.prefix}-backend"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

resource "aws_ecs_service" "fleet" {
  name                               = "fleet"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.fleet.id
  task_definition                    = aws_ecs_task_definition.backend.arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 30

  load_balancer {
    target_group_arn = aws_alb_target_group.main.arn
    container_name   = "fleet"
    container_port   = 8080
  }

  network_configuration {
    subnets         = module.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }

  depends_on = [aws_alb_listener.http]
}

resource "aws_cloudwatch_log_group" "backend" {
  name              = "fleetdm"
  retention_in_days = 1
}

data "aws_region" "current" {}


resource "aws_ecs_task_definition" "backend" {
  family                   = "fleet"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode(
    [
      {
        name        = "fleet"
        image       = "fleetdm/fleet"
        cpu         = 256
        memory      = 512
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
            awslogs-group         = aws_cloudwatch_log_group.backend.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "fleet"
          }
        },
        secrets = [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          }
        ]
        environment = [
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = "fleet"
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = "fleet"
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_endpoint}:3306"
          },
          {
            name  = "FLEET_REDIS_ADDRESS"
            value = "${aws_elasticache_replication_group.default.primary_endpoint_address}:6379"
          },
          {
            name  = "FLEET_FIREHOSE_STATUS_STREAM"
            value = aws_kinesis_firehose_delivery_stream.osquery_logs.name
          },
          {
            name  = "FLEET_FIREHOSE_RESULT_STREAM"
            value = aws_kinesis_firehose_delivery_stream.osquery_logs.name
          },
          {
            name  = "FLEET_FIREHOSE_REGION"
            value = data.aws_region.current.name
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
          }
        ]
      }
  ])
}

resource "aws_ecs_task_definition" "migration" {
  family                   = "fleet-migrate"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode(
    [
      {
        name        = "fleet-prepare-db"
        image       = "fleetdm/fleet"
        cpu         = 256
        memory      = 512
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
            awslogs-group         = aws_cloudwatch_log_group.backend.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "fleet"
          }
        },
        command = ["fleet", "prepare", "db"]
        secrets = [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          }
        ]
        environment = [
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = "fleet"
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = "fleet"
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_endpoint}:3306"
          },
          {
            name  = "FLEET_REDIS_ADDRESS"
            value = "${aws_elasticache_replication_group.default.primary_endpoint_address}:6379"
          }
        ]
      }
  ])
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = 5
  min_capacity       = 1
  resource_id        = "service/${aws_ecs_cluster.fleet.name}/${aws_ecs_service.fleet.name}"
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
    target_value = 80
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

    target_value = 60
  }
}

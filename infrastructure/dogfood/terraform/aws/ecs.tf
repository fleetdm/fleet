data "aws_region" "current" {}

resource "aws_route53_record" "record" {
  name    = "fleet-alb-${terraform.workspace}"
  type    = "A"
  zone_id = aws_route53_zone.dogfood_fleetdm_com.zone_id
  alias {
    evaluate_target_health = false
    name                   = aws_alb.main.dns_name
    zone_id                = aws_alb.main.zone_id
  }
}

resource "aws_alb" "main" {
  // Exposed to the Internet by design
  internal                   = false #tfsec:ignore:aws-elb-alb-not-public
  security_groups            = [aws_security_group.lb.id, aws_security_group.backend.id]
  subnets                    = module.vpc.public_subnets
  idle_timeout               = 905
  name                       = "fleetdm"
  drop_invalid_header_fields = true
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

resource "aws_alb_listener" "https-fleetdm" {
  load_balancer_arn = aws_alb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate_validation.dogfood_fleetdm_com.certificate_arn

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
  desired_count                      = 5
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 30

  load_balancer {
    target_group_arn = aws_alb_target_group.main.arn
    container_name   = "fleet"
    container_port   = 8080
  }

  // https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service#ignoring-changes-to-desired-count
  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = module.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }

  depends_on = [aws_alb_listener.http, aws_alb_listener.https-fleetdm]
}
// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future.
resource "aws_cloudwatch_log_group" "backend" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key:exp:2022-07-01
  name              = "fleetdm"
  retention_in_days = var.cloudwatch_log_retention
}

resource "aws_ecs_task_definition" "backend" {
  family                   = "fleet"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = var.fleet_backend_cpu
  memory                   = var.fleet_backend_mem
  container_definitions = jsonencode(
    [
      {
        name        = "fleet"
        image       = var.fleet_image
        cpu         = var.fleet_backend_cpu
        memory      = var.fleet_backend_mem
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
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          },
          {
            name      = "FLEET_MYSQL_READ_REPLICA_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          }
        ]
        environment = [
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = var.database_user
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = var.database_name
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_endpoint}:3306"
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_USERNAME"
            value = var.database_user
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_DATABASE"
            value = var.database_name
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_reader_endpoint}:3306"
          },
          {
            name  = "FLEET_REDIS_ADDRESS"
            value = "${aws_elasticache_replication_group.default.primary_endpoint_address}:6379"
          },
          {
            name  = "FLEET_REDIS_USE_TLS"
            value = "true"
          },
          {
            name  = "FLEET_FIREHOSE_STATUS_STREAM"
            value = aws_kinesis_firehose_delivery_stream.osquery_status.name
          },
          {
            name  = "FLEET_FIREHOSE_RESULT_STREAM"
            value = aws_kinesis_firehose_delivery_stream.osquery_results.name
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
          },
          {
            name  = "FLEET_VULNERABILITIES_DATABASES_PATH"
            value = var.vuln_db_path
          },
          {
            name  = "FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING"
            value = var.async_host_processing
          },
          {
            name  = "FLEET_LOGGING_DEBUG"
            value = var.logging_debug
          },
          {
            name  = "FLEET_LOGGING_JSON"
            value = var.logging_json
          },
          {
            name  = "FLEET_S3_BUCKET"
            value = aws_s3_bucket.osquery-carve.bucket
          },
          {
            name  = "FLEET_S3_PREFIX"
            value = "carve_results/"
          },
          {
            name  = "FLEET_LICENSE_KEY"
            value = var.fleet_license
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
  cpu                      = var.cpu_migrate
  memory                   = var.mem_migrate
  container_definitions = jsonencode(
    [
      {
        name        = "fleet-prepare-db"
        image       = var.fleet_image
        cpu         = var.cpu_migrate
        memory      = var.mem_migrate
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
        command = ["fleet", "prepare", "--no-prompt=true", "db"]
        secrets = [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          }
        ]
        environment = [
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = var.database_user
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = var.database_name
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
            name  = "FLEET_REDIS_USE_TLS"
            value = "true"
          }
        ]
      }
  ])
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = var.fleet_max_capacity
  min_capacity       = var.fleet_min_capacity
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
    target_value = var.memory_tracking_target_value
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

    target_value = var.cpu_tracking_target_value
  }
}

output "fleet_ecs_cluster_arn" {
  value = aws_ecs_cluster.fleet.arn
}

output "fleet_ecs_cluster_id" {
  value = aws_ecs_cluster.fleet.id
}

resource "aws_alb" "main" {
  name                       = "fleetdm"
  internal                   = false #tfsec:ignore:aws-elb-alb-not-public
  security_groups            = [aws_security_group.lb.id, aws_security_group.backend.id]
  subnets                    = module.vpc.public_subnets
  idle_timeout               = 600
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it
}

resource "aws_alb" "internal" {
  name                       = "fleetdm-internal"
  internal                   = true
  security_groups            = [aws_security_group.lb.id, aws_security_group.backend.id]
  subnets                    = module.vpc.private_subnets
  idle_timeout               = 600
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it
}

resource "aws_alb_listener" "https-fleetdm-internal" {
  load_balancer_arn = aws_alb.internal.arn
  port              = 80
  protocol          = "HTTP" #tfsec:ignore:aws-elb-http-not-used

  default_action {
    target_group_arn = aws_alb_target_group.internal.arn
    type             = "forward"
  }
}

resource "aws_alb_target_group" "internal" {
  name                 = "fleetdm-internal"
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
  name = "${local.prefix}-backend"

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
  desired_count                      = var.scale_down ? 0 : 10
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 30

  load_balancer {
    target_group_arn = aws_alb_target_group.internal.arn
    container_name   = "fleet"
    container_port   = 8080
  }

  load_balancer {
    target_group_arn = aws_alb_target_group.main.arn
    container_name   = "fleet"
    container_port   = 8080
  }

  network_configuration {
    subnets         = module.vpc.private_subnets
    security_groups = [aws_security_group.backend.id]
  }

  depends_on = [aws_alb_listener.http, aws_alb_listener.https-fleetdm]
}

resource "aws_cloudwatch_log_group" "backend" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key
  name              = "fleetdm"
  retention_in_days = 1
}

data "aws_region" "current" {}

data "aws_secretsmanager_secret" "license" {
  name = "/fleet/license"
}

resource "aws_ecs_task_definition" "backend" {
  family                   = "fleet"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 1024
  memory                   = 4096
  container_definitions = jsonencode(
    [
      {
        name      = "prometheus-exporter"
        image     = "917007347864.dkr.ecr.us-east-2.amazonaws.com/prometheus-to-cloudwatch:latest"
        essential = false
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.backend.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "fleet-prometheus-exporter"
          }
        }
        environment = [
          {
            name  = "CLOUDWATCH_NAMESPACE"
            value = "fleet-loadtest"
          },
          {
            name  = "CLOUDWATCH_REGION"
            value = "us-east-2"
          },
          {
            name  = "PROMETHEUS_SCRAPE_URL"
            value = "http://localhost:8080/metrics"
          },
        ],
      },
      {
        name        = "fleet"
        image       = docker_registry_image.fleet.name
        cpu         = 1024
        memory      = 4096
        mountPoints = []
        volumesFrom = []
        essential   = true
        portMappings = [
          {
            # This port is the same that the contained application also uses
            containerPort = 8080
            hostPort      = 8080
            protocol      = "tcp"
          }
        ]
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
            awslogs-stream-prefix = "fleet"
          }
        },
        secrets = [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          },
          {
            name      = "FLEET_MYSQL_READ_REPLICA_PASSWORD"
            valueFrom = aws_secretsmanager_secret.database_password_secret.arn
          },
          {
            name      = "FLEET_LICENSE_KEY"
            valueFrom = data.aws_secretsmanager_secret.license.arn
          }
        ]
        environment = concat([
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = module.aurora_mysql.rds_cluster_master_username
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = module.aurora_mysql.rds_cluster_database_name
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_endpoint}:3306"
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_USERNAME"
            value = module.aurora_mysql.rds_cluster_master_username
          },
          {
            name  = "FLEET_MYSQL_READ_REPLICA_DATABASE"
            value = module.aurora_mysql.rds_cluster_database_name
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
            name  = "FLEET_REDIS_CLUSTER_FOLLOW_REDIRECTIONS"
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
            name  = "FLEET_REDIS_MAX_IDLE_CONNS"
            value = "100"
          },
          {
            name  = "FLEET_REDIS_MAX_OPEN_CONNS"
            value = "100"
          },
          {
            name  = "FLEET_OSQUERY_ASYNC_HOST_REDIS_SCAN_KEYS_COUNT"
            value = "10000"
          }
        ], local.additional_env_vars)
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}


resource "aws_ecs_task_definition" "migration" {
  family                   = "fleet-migrate"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 1024
  memory                   = 2048
  container_definitions = jsonencode(
    [
      {
        name        = "fleet-prepare-db"
        image       = docker_registry_image.fleet.name
        cpu         = 1024
        memory      = 2048
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
            awslogs-stream-prefix = "fleet-migration"
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
            name  = "CLOUDWATCH_NAMESPACE"
            value = "fleet-loadtest-migration"
          },
          {
            name  = "CLOUDWATCH_REGION"
            value = "us-east-2"
          },
          {
            name  = "FLEET_MYSQL_USERNAME"
            value = module.aurora_mysql.rds_cluster_master_username
          },
          {
            name  = "FLEET_MYSQL_DATABASE"
            value = module.aurora_mysql.rds_cluster_database_name
          },
          {
            name  = "FLEET_MYSQL_ADDRESS"
            value = "${module.aurora_mysql.rds_cluster_endpoint}:3306"
          },
          {
            name  = "FLEET_REDIS_ADDRESS"
            value = "${aws_elasticache_replication_group.default.primary_endpoint_address}:6379"
          },
        ]
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = var.scale_down ? 0 : 10
  min_capacity       = var.scale_down ? 0 : 10
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

    target_value = 90
  }
}

output "fleet_migration_revision" {
  value = aws_ecs_task_definition.migration.revision
}

output "fleet_migration_subnets" {
  value = jsonencode(aws_ecs_service.fleet.network_configuration[0].subnets)
}

output "fleet_migration_security_groups" {
  value = jsonencode(aws_ecs_service.fleet.network_configuration[0].security_groups)
}

output "fleet_ecs_cluster_arn" {
  value = aws_ecs_cluster.fleet.arn
}

output "fleet_ecs_cluster_id" {
  value = aws_ecs_cluster.fleet.id
}

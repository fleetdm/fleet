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
  idle_timeout    = 600
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
  desired_count                      = 10
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

  depends_on = [aws_alb_listener.http, aws_alb_listener.https-fleetdm]
}

resource "aws_cloudwatch_log_group" "backend" {
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
  cpu                      = var.fleet_backend_cpu
  memory                   = var.fleet_backend_mem
  container_definitions = jsonencode(
    [
      {
        name      = "cloudwatch-agent"
        image     = "amazon/cloudwatch-agent:1.247348.0b251302"
        essential = false
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.backend.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "fleet-cw-agent"
          }
        }
        environment = [
          {
            name  = "PROMETHEUS_CONFIG_CONTENT"
            value = <<-EOT
            global:
              scrape_interval: 1m
              scrape_timeout: 10s
            scrape_configs:
              - job_name: cwagent-ecs-file-sd-config
                sample_limit: 10000
                file_sd_configs:
                  - files: [ "/tmp/cwagent_ecs_auto_sd.yaml" ]
            EOT
          },
          {
            name  = "CW_CONFIG_CONTENT"
            value = <<-EOT
            {
              "logs": {
                "metrics_collected": {
                  "prometheus": {
                    "prometheus_config_path": "env:PROMETHEUS_CONFIG_CONTENT",
                    "ecs_service_discovery": {
                      "sd_frequency": "1m",
                      "sd_result_file": "/tmp/cwagent_ecs_auto_sd.yaml",
                      "docker_label": {
                      },
                      "task_definition_list": [
                        {
                          "sd_job_name": "ecs-appmesh-colors",
                          "sd_metrics_ports": "9901",
                          "sd_task_definition_arn_pattern": ".*:task-definition/.*-ColorTeller-(white):[0-9]+",
                          "sd_metrics_path": "/stats/prometheus"
                        },
                        {
                          "sd_job_name": "ecs-appmesh-gateway",
                          "sd_metrics_ports": "9901",
                          "sd_task_definition_arn_pattern": ".*:task-definition/.*-ColorGateway:[0-9]+",
                          "sd_metrics_path": "/stats/prometheus"
                        }
                      ]
                    },
                    "emf_processor": {
                      "metric_declaration_dedup": true,
                      "metric_declaration": [
                        {
                          "source_labels": ["container_name"],
                          "label_matcher": "^envoy$",
                          "dimensions": [["ClusterName","TaskDefinitionFamily"]],
                          "metric_selectors": [
                            "^envoy_http_downstream_rq_(total|xx)$",
                            "^envoy_cluster_upstream_cx_(r|t)x_bytes_total$",
                            "^envoy_cluster_membership_(healthy|total)$",
                            "^envoy_server_memory_(allocated|heap_size)$",
                            "^envoy_cluster_upstream_cx_(connect_timeout|destroy_local_with_active_rq)$",
                            "^envoy_cluster_upstream_rq_(pending_failure_eject|pending_overflow|timeout|per_try_timeout|rx_reset|maintenance_mode)$",
                            "^envoy_http_downstream_cx_destroy_remote_active_rq$",
                            "^envoy_cluster_upstream_flow_control_(paused_reading_total|resumed_reading_total|backed_up_total|drained_total)$",
                            "^envoy_cluster_upstream_rq_retry$",
                            "^envoy_cluster_upstream_rq_retry_(success|overflow)$",
                            "^envoy_server_(version|uptime|live)$"
                          ]
                        },
                        {
                          "source_labels": ["container_name"],
                          "label_matcher": "^envoy$",
                          "dimensions": [["ClusterName","TaskDefinitionFamily","envoy_http_conn_manager_prefix","envoy_response_code_class"]],
                          "metric_selectors": [
                            "^envoy_http_downstream_rq_xx$"
                          ]
                        },
                        {
                          "source_labels": ["Java_EMF_Metrics"],
                          "label_matcher": "^true$",
                          "dimensions": [["ClusterName","TaskDefinitionFamily"]],
                          "metric_selectors": [
                            "^jvm_threads_(current|daemon)$",
                            "^jvm_classes_loaded$",
                            "^java_lang_operatingsystem_(freephysicalmemorysize|totalphysicalmemorysize|freeswapspacesize|totalswapspacesize|systemcpuload|processcpuload|availableprocessors|openfiledescriptorcount)$",
                            "^catalina_manager_(rejectedsessions|activesessions)$",
                            "^jvm_gc_collection_seconds_(count|sum)$",
                            "^catalina_globalrequestprocessor_(bytesreceived|bytessent|requestcount|errorcount|processingtime)$"
                          ]
                        },
                        {
                          "source_labels": ["Java_EMF_Metrics"],
                          "label_matcher": "^true$",
                          "dimensions": [["ClusterName","TaskDefinitionFamily","area"]],
                          "metric_selectors": [
                            "^jvm_memory_bytes_used$"
                          ]
                        },
                        {
                          "source_labels": ["Java_EMF_Metrics"],
                          "label_matcher": "^true$",
                          "dimensions": [["ClusterName","TaskDefinitionFamily","pool"]],
                          "metric_selectors": [
                            "^jvm_memory_pool_bytes_used$"
                          ]
                        }
                      ]
                    }
                  }
                },
                "force_flush_interval": 5
              }
            }
            EOT
          }
        ]
      },
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
            name  = "FLEET_BETA_SOFTWARE_INVENTORY"
            value = var.software_inventory
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

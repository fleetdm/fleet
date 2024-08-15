locals {
  environment = [for k, v in var.fleet_config.extra_environment_variables : {
    name  = k
    value = v
  }]
  secrets = [for k, v in var.fleet_config.extra_secrets : {
    name      = k
    valueFrom = v
  }]
  load_balancers = concat([
    {
      target_group_arn = var.fleet_config.loadbalancer.arn
      container_name   = "fleet"
      container_port   = 8080
    }
  ], var.fleet_config.extra_load_balancers)
  repository_credentials = var.fleet_config.repository_credentials != "" ? {
    repositoryCredentials = {
      credentialsParameter = var.fleet_config.repository_credentials
    }
  } : null
}

data "aws_region" "current" {}

resource "aws_ecs_service" "fleet" {
  name                               = var.fleet_config.service.name
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.backend.arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 30

  dynamic "load_balancer" {
    for_each = local.load_balancers
    content {
      target_group_arn = load_balancer.value.target_group_arn
      container_name   = load_balancer.value.container_name
      container_port   = load_balancer.value.container_port
    }
  }

  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = var.fleet_config.networking.subnets
    security_groups = var.fleet_config.networking.security_groups == null ? aws_security_group.main.*.id : var.fleet_config.networking.security_groups
  }
}

resource "aws_ecs_task_definition" "backend" {
  family                   = var.fleet_config.family
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = var.fleet_config.iam_role_arn == null ? aws_iam_role.main[0].arn : var.fleet_config.iam_role_arn
  execution_role_arn       = aws_iam_role.execution.arn
  cpu                      = var.fleet_config.task_cpu == null ? var.fleet_config.cpu : var.fleet_config.task_cpu
  memory                   = var.fleet_config.task_mem == null ? var.fleet_config.mem : var.fleet_config.task_mem
  pid_mode                 = var.fleet_config.pid_mode
  container_definitions = jsonencode(
    concat([
      {
        name        = "fleet"
        image       = var.fleet_config.image
        cpu         = var.fleet_config.cpu
        memory      = var.fleet_config.mem
        mountPoints = var.fleet_config.mount_points
        dependsOn   = var.fleet_config.depends_on
        volumesFrom = []
        essential   = true
        portMappings = [
          {
            # This port is the same that the contained application also uses
            containerPort = 8080
            protocol      = "tcp"
          }
        ]
        repositoryCredentials = local.repository_credentials
        networkMode           = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = var.fleet_config.awslogs.create == true ? aws_cloudwatch_log_group.main[0].name : var.fleet_config.awslogs.name
            awslogs-region        = var.fleet_config.awslogs.create == true ? data.aws_region.current.name : var.fleet_config.awslogs.region
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
        secrets = concat([
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = var.fleet_config.database.password_secret_arn
          },
          {
            name      = "FLEET_MYSQL_READ_REPLICA_PASSWORD"
            valueFrom = var.fleet_config.database.password_secret_arn
          },
          {
            name      = "FLEET_SERVER_PRIVATE_KEY"
            valueFrom = aws_secretsmanager_secret.fleet_server_private_key.arn
          }
        ], local.secrets)
        environment = concat([
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
          },
          {
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
            name  = "FLEET_SERVER_TLS"
            value = "false"
          },
          {
            name  = "FLEET_S3_SOFTWARE_INSTALLERS_BUCKET"
            value = var.fleet_config.software_installers.create_bucket == true ? aws_s3_bucket.software_installers[0].bucket : var.fleet_config.software_installers.bucket_name
          },
          {
            name  = "FLEET_S3_SOFTWARE_INSTALLERS_PREFIX"
            value = var.fleet_config.software_installers.s3_object_prefix
          },
        ], local.environment)
      }
  ], var.fleet_config.sidecars))
  dynamic "volume" {
    for_each = var.fleet_config.volumes
    content {
      name      = volume.value.name
      host_path = lookup(volume.value, "host_path", null)

      dynamic "docker_volume_configuration" {
        for_each = lookup(volume.value, "docker_volume_configuration", [])
        content {
          scope         = lookup(docker_volume_configuration.value, "scope", null)
          autoprovision = lookup(docker_volume_configuration.value, "autoprovision", null)
          driver        = lookup(docker_volume_configuration.value, "driver", null)
          driver_opts   = lookup(docker_volume_configuration.value, "driver_opts", null)
          labels        = lookup(docker_volume_configuration.value, "labels", null)
        }
      }

      dynamic "efs_volume_configuration" {
        for_each = lookup(volume.value, "efs_volume_configuration", [])
        content {
          file_system_id = lookup(efs_volume_configuration.value, "file_system_id", null)
          root_directory = lookup(efs_volume_configuration.value, "root_directory", null)
        }
      }
    }
  }
}

resource "aws_appautoscaling_target" "ecs_target" {
  max_capacity       = var.fleet_config.autoscaling.max_capacity
  min_capacity       = var.fleet_config.autoscaling.min_capacity
  resource_id        = "service/${var.ecs_cluster}/${aws_ecs_service.fleet.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "ecs_policy_memory" {
  name               = "${var.fleet_config.family}-memory-autoscaling"
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
  name               = "${var.fleet_config.family}-cpu-autoscaling"
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
  count             = var.fleet_config.awslogs.create == true ? 1 : 0
  name              = var.fleet_config.awslogs.name
  retention_in_days = var.fleet_config.awslogs.retention
}

resource "aws_security_group" "main" {
  count       = var.fleet_config.networking.security_groups == null ? 1 : 0
  name        = var.fleet_config.security_group_name
  description = "Fleet ECS Service Security Group"
  vpc_id      = var.vpc_id
  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  ingress {
    description      = "Ingress only on container port"
    from_port        = 8080
    to_port          = 8080
    protocol         = "TCP"
    cidr_blocks      = var.fleet_config.networking.ingress_sources.cidr_blocks
    ipv6_cidr_blocks = var.fleet_config.networking.ingress_sources.ipv6_cidr_blocks
    security_groups  = var.fleet_config.networking.ingress_sources.security_groups
    prefix_list_ids  = var.fleet_config.networking.ingress_sources.prefix_list_ids
  }
}

resource "random_password" "fleet_server_private_key" {
  length  = 32
  special = true
}

resource "aws_secretsmanager_secret" "fleet_server_private_key" {
  name = var.fleet_config.private_key_secret_name

  recovery_window_in_days = "0"
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_secretsmanager_secret_version" "fleet_server_private_key" {
  secret_id     = aws_secretsmanager_secret.fleet_server_private_key.id
  secret_string = random_password.fleet_server_private_key.result
}

// Customer keys are not supported in our Fleet Terraforms at the moment. We will evaluate the
// possibility of providing this capability in the future.
// No versioning on this bucket is by design.
// Bucket logging is not supported in our Fleet Terraforms at the moment. It can be enabled by the
// organizations deploying Fleet, and we will evaluate the possibility of providing this capability
// in the future.

resource "aws_s3_bucket" "software_installers" { #tfsec:ignore:aws-s3-encryption-customer-key:exp:2022-07-01  #tfsec:ignore:aws-s3-enable-versioning #tfsec:ignore:aws-s3-enable-bucket-logging:exp:2022-06-15
  count         = var.fleet_config.software_installers.create_bucket == true ? 1 : 0
  bucket        = var.fleet_config.software_installers.bucket_name
  bucket_prefix = var.fleet_config.software_installers.bucket_prefix
}

resource "aws_s3_bucket_server_side_encryption_configuration" "software_installers" {
  count  = var.fleet_config.software_installers.create_bucket == true ? 1 : 0
  bucket = aws_s3_bucket.software_installers[0].bucket
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "aws:kms"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "software_installers" {
  count                   = var.fleet_config.software_installers.create_bucket == true ? 1 : 0
  bucket                  = aws_s3_bucket.software_installers[0].id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

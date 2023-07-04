locals {
  environment = [for k, v in var.fleet_config.extra_environment_variables : {
    name  = k
    value = v
  }]
  secrets = [for k, v in var.fleet_config.extra_secrets : {
    name      = k
    valueFrom = v
  }]
}

data "aws_region" "current" {}

resource "aws_cloudwatch_log_group" "main" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key:exp:2022-07-01
  count             = var.fleet_config.awslogs.create == true ? 1 : 0
  name              = var.fleet_config.awslogs.name
  retention_in_days = var.fleet_config.awslogs.retention
}

resource "aws_ecs_task_definition" "vuln-data-stream" {
  family                   = var.fleet_config.family
  cpu                      = var.fleet_config.vuln_data_stream_cpu
  memory                   = var.fleet_config.vuln_data_stream_mem
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.main.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]

  container_definitions = jsonencode([
    {
      name        = "fleet-vuln-provisioner"
      image       = var.fleet_config.image
      essential   = true
      user        = "root"
      command     = ["fleetctl", "vulnerability-data-stream", "--dir=${var.fleet_config.vuln_database_path}"]
      networkMode = "awsvpc"
      mountPoints = [
        {
          sourceVolume  = "efs-mount"
          containerPath = var.fleet_config.vuln_database_path
          readOnly      = false
        }
      ],
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.fleet_config.awslogs.create == true ? aws_cloudwatch_log_group.main[0].name : var.fleet_config.awslogs.name
          awslogs-region        = var.fleet_config.awslogs.create == true ? data.aws_region.current.name : var.fleet_config.awslogs.region
          awslogs-stream-prefix = "${var.fleet_config.awslogs.prefix}-data-stream"
        }
      }
    }
  ])

  volume {
    name = "efs-mount"
    efs_volume_configuration {
      file_system_id = aws_efs_file_system.vuln.id
      root_directory = var.efs_root_directory
    }
  }
}


resource "aws_ecs_task_definition" "vuln-processing" {
  family                   = var.fleet_config.family
  cpu                      = var.fleet_config.vuln_processing_cpu
  memory                   = var.fleet_config.vuln_processing_mem
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.main.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]

  container_definitions = jsonencode([
    {
      name        = "fleet-vuln-processing"
      image       = var.fleet_config.image
      essential   = true
      command     = ["fleet", "vuln_processing"]
      user        = "root"
      networkMode = "awsvpc"
      mountPoints = [
        {
          sourceVolume  = "efs-mount"
          containerPath = var.fleet_config.vuln_database_path
          readOnly      = false
        }
      ],
      secrets = concat(
        [
          {
            name      = "FLEET_MYSQL_PASSWORD"
            valueFrom = var.fleet_config.database.password_secret_arn
          }
      ], local.secrets),
      environment = concat(
        [
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
            name  = "FLEET_VULNERABILITIES_DISABLE_DATA_SYNC"
            value = "true"
          },
          {
            name  = "FLEET_VULNERABILITIES_DATABASES_PATH"
            value = var.fleet_config.vuln_database_path
          }
      ], local.environment),
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.fleet_config.awslogs.create == true ? aws_cloudwatch_log_group.main[0].name : var.fleet_config.awslogs.name
          awslogs-region        = var.fleet_config.awslogs.create == true ? data.aws_region.current.name : var.fleet_config.awslogs.region
          awslogs-stream-prefix = "${var.fleet_config.awslogs.prefix}-procssing"
        }
      }
    }
  ])

  volume {
    name = "efs-mount"
    efs_volume_configuration {
      file_system_id = aws_efs_file_system.vuln.id
      root_directory = var.efs_root_directory
    }
  }
}

resource "aws_cloudwatch_event_rule" "vuln_processing" {
  name_prefix         = "${var.customer_prefix}-vuln-processing"
  schedule_expression = var.fleet_config.vuln_processing_schedule_expression
}

resource "aws_cloudwatch_event_target" "vuln_processing" {
  arn      = var.ecs_cluster
  rule     = aws_cloudwatch_event_rule.vuln_processing.name
  role_arn = aws_iam_role.run_cloudwatch.arn
  ecs_target {
    task_definition_arn = aws_ecs_task_definition.vuln-processing.arn
    task_count          = 1
    launch_type         = "FARGATE"

    network_configuration {
      assign_public_ip = false
      subnets          = var.fleet_config.networking.subnets
      security_groups  = var.fleet_config.networking.security_groups
    }
  }
}

resource "aws_cloudwatch_event_rule" "vuln_data_stream" {
  name_prefix         = "${var.customer_prefix}-vuln-data-stream"
  schedule_expression = var.fleet_config.vuln_data_stream_schedule_expression
}

resource "aws_cloudwatch_event_target" "vuln_data_stream" {
  arn      = var.ecs_cluster
  rule     = aws_cloudwatch_event_rule.vuln_data_stream.name
  role_arn = aws_iam_role.run_cloudwatch.arn
  ecs_target {
    task_definition_arn = aws_ecs_task_definition.vuln-data-stream.arn
    task_count          = 1
    launch_type         = "FARGATE"
    network_configuration {
      assign_public_ip = false
      subnets          = var.fleet_config.networking.subnets
      security_groups  = var.fleet_config.networking.security_groups
    }
  }
}


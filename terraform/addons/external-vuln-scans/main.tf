data "aws_region" "current" {}

locals {
  environment = [
    // specifically overriding disable schedule here because the output of this module sets this to true
    // and then we pull in the output of fleet ecs module
    for k, v in merge(
      var.fleet_config.extra_environment_variables,
      { FLEET_VULNERABILITIES_DISABLE_SCHEDULE = "false" }
      ) : {
      name  = k
      value = v
    }
  ]
  secrets = [
    for k, v in merge(var.fleet_config.extra_secrets, {
      FLEET_MYSQL_PASSWORD = var.fleet_config.database.password_secret_arn
      }) : {
      name      = k
      valueFrom = v
    }
  ]
  repository_credentials = var.fleet_config.repository_credentials != "" ? {
    repositoryCredentials = {
      credentialsParameter = var.fleet_config.repository_credentials
    }
  } : {}
}

resource "aws_ecs_service" "fleet" {
  name                               = "${var.fleet_config.service.name}-vuln-processing"
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.vuln-processing.arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = var.subnets
    security_groups = var.security_groups
  }
}

resource "aws_ecs_task_definition" "vuln-processing" {
  family                   = "${var.fleet_config.family}-vuln-processing"
  cpu                      = var.vuln_processing_cpu
  memory                   = var.vuln_processing_memory
  execution_role_arn       = var.execution_iam_role_arn
  task_role_arn            = var.task_role_arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]

  container_definitions = jsonencode(concat([
    {
      name                  = "fleet-vuln-processing"
      image                 = var.fleet_config.image
      essential             = true
      networkMode           = "awsvpc"
      secrets               = local.secrets
      repositoryCredentials = local.repository_credentials
      ulimits = [
        {
          name      = "nofile"
          softLimit = 999999
          hardLimit = 999999
        }
      ],
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
      ], local.environment),
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.awslogs_config.group
          awslogs-region        = var.awslogs_config.region == null ? data.aws_region.current.name : var.awslogs_config.region
          awslogs-stream-prefix = "${var.awslogs_config.prefix}-vuln-processing"
        }
      }
    }]
  , var.fleet_config.sidecars))
}




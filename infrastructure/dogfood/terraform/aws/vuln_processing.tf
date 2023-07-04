resource "aws_ecs_task_definition" "vuln-processing" {
  family                   = "fleet-vuln-processing"
  cpu                      = 2048
  memory                   = 4096
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]

  container_definitions = jsonencode([
    {
      name        = "fleet-vuln-processing"
      image       = var.fleet_image
      essential   = true
      command     = ["fleet", "vuln_processing"]
      networkMode = "awsvpc"
      secrets = [
        {
          name      = "FLEET_MYSQL_PASSWORD"
          valueFrom = aws_secretsmanager_secret.database_password_secret.arn
        }
      ]
      environment = [
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
          name  = "FLEET_VULNERABILITIES_DATABASES_PATH"
          value = "/home/fleet/vuln_data"
        },
        {
          name  = "FLEET_LOGGING_DEBUG"
          value = "true"
        },
        {
          name  = "FLEET_LICENSE_KEY"
          value = var.fleet_license
        }
      ],
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.backend.name
          awslogs-region        = data.aws_region.current.name
          awslogs-stream-prefix = "fleet-vuln-processing"
        }
      }
    }
  ])
}

resource "aws_cloudwatch_event_rule" "vuln_processing" {
  name_prefix         = "${local.name}-vuln-processing"
  schedule_expression = "rate(1 hour)"
  is_enabled          = false
}

resource "aws_cloudwatch_event_target" "vuln_processing" {
  arn      = aws_ecs_cluster.fleet.arn
  rule     = aws_cloudwatch_event_rule.vuln_processing.name
  role_arn = aws_iam_role.run_cloudwatch.arn
  ecs_target {
    task_definition_arn = aws_ecs_task_definition.vuln-processing.arn
    task_count          = 1
    launch_type         = "FARGATE"
    network_configuration {
      assign_public_ip = false
      subnets          = module.vpc.private_subnets
      security_groups  = [aws_security_group.backend.id]
    }
  }
}


data "aws_iam_policy_document" "assume_events" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }
  }
}



data "aws_iam_policy_document" "cloudwatch_task" {
  statement {
    effect    = "Allow"
    actions   = ["iam:PassRole"]
    resources = ["*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["ecs:RunTask"]
    resources = ["*"]
    condition {
      test     = "ArnEquals"
      variable = "ecs:cluster"
      values   = [aws_ecs_cluster.fleet.arn]
    }
  }
}

data "aws_iam_policy_document" "assume_role_policy" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "run_cloudwatch" {
  name               = "${local.name}-cloudwatch-run"
  assume_role_policy = data.aws_iam_policy_document.assume_events.json
}

resource "aws_iam_policy" "run_cloudwatch" {
  name   = "${local.name}-cloudwatch-run"
  policy = data.aws_iam_policy_document.cloudwatch_task.json
}
resource "aws_iam_role_policy_attachment" "run_cloudwatch" {
  role       = aws_iam_role.run_cloudwatch.name
  policy_arn = aws_iam_policy.run_cloudwatch.arn
}

resource "aws_iam_role_policy_attachment" "ecs_role_attachment" {
  role       = aws_iam_role.main.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceRole"
}

resource "aws_iam_role_policy_attachment" "ecs_task" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceEventsRole"
  role       = aws_iam_role.main.name
}
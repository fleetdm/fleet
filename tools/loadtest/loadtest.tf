data "aws_iam_policy_document" "fleet" {
  statement {
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.database_password_secret.arn, data.aws_secretsmanager_secret.license.arn]
  }

  statement {
    effect = "Allow"
    actions = [
      "firehose:DescribeDeliveryStream",
      "firehose:PutRecord",
      "firehose:PutRecordBatch",
    ]
    resources = [aws_kinesis_firehose_delivery_stream.osquery_logs.arn]
  }
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["ecs.amazonaws.com", "ecs-tasks.amazonaws.com"]
      type        = "Service"
    }
  }
}

resource "aws_iam_role" "main" {
  name               = "fleetdm-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "role_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.main.name
}

resource "aws_iam_policy" "main" {
  name   = "fleet-iam-policy"
  policy = data.aws_iam_policy_document.fleet.json
}

resource "aws_iam_role_policy_attachment" "attachment" {
  policy_arn = aws_iam_policy.main.arn
  role       = aws_iam_role.main.name
}

#--

data "aws_region" "current" {}

resource "aws_ecs_cluster" "osquery-perf" {
  name = "${var.prefix}-osquery-perf"
}

resource "aws_ecs_service" "osquery-perf" {
  name                               = "osquery-perf"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.osquery-perf.id
  task_definition                    = aws_ecs_task_definition.osquery-perf.arn
  desired_count                      = var.osquery_host_count

  network_configuration {
    subnets         = module.vpc.private_subnets
#    security_groups = [aws_security_group.backend.id]
  }
}


resource "aws_ecs_task_definition" "osquery-perf" {
  family                   = "osquery-perf"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.main.arn
  task_role_arn            = aws_iam_role.main.arn
  cpu                      = 2048
  memory                   = 4096
  container_definitions = jsonencode(
  [
    {
      name        = "osquery-perf"
      image       = "917007347864.dkr.ecr.us-east-2.amazonaws.com/osquery-perf"
      cpu         = 2048
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
  max_capacity       = var.osquery_host_count
  min_capacity       = var.osquery_host_count
  resource_id        = "service/${aws_ecs_cluster.osquery-perf.name}/${aws_ecs_service.osquery-perf.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_cloudwatch_log_group" "osquery-perf" {
  name              = "osquery-perf"
  retention_in_days = 1
}
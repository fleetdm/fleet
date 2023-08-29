data "aws_region" "current" {}

resource "aws_cloudwatch_event_rule" "main" {
  schedule_expression = "rate(1 hour)"
}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["events.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

resource "aws_iam_role" "ecs_events" {
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

data "aws_iam_policy_document" "ecs_events_run_task_with_any_role" {
  statement {
    effect    = "Allow"
    actions   = ["iam:PassRole"]
    resources = ["*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["ecs:RunTask"]
    resources = [replace(var.task_definition.arn, "/:\\d+$/", ":*")]
  }
}
resource "aws_iam_role_policy" "ecs_events_run_task_with_any_role" {
  role   = aws_iam_role.ecs_events.id
  policy = data.aws_iam_policy_document.ecs_events_run_task_with_any_role.json
}

resource "aws_cloudwatch_event_target" "ecs_scheduled_task" {
  arn      = var.ecs_cluster.cluster_arn
  rule     = aws_cloudwatch_event_rule.main.name
  role_arn = aws_iam_role.ecs_events.arn

  ecs_target {
    task_count          = 1
    task_definition_arn = var.task_definition.arn
    launch_type         = "FARGATE"
    network_configuration {
      subnets         = var.ecs_service.network_configuration[0].subnets
      security_groups = var.ecs_service.network_configuration[0].security_groups
    }
  }

  input = jsonencode({
    containerOverrides = [
      {
        name    = "fleet",
        command = ["fleet", "vuln_processing"]
      },
      {
        resourceRequirements = [
          {
            type  = "VCPU",
            value = "1"
          },
          {
            type  = "MEMORY",
            value = "4096"
          }
        ]
      }
    ]
  })
}

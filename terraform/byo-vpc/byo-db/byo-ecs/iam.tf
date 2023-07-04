data "aws_iam_policy_document" "fleet" {
  statement {
    effect    = "Allow"
    actions   = ["cloudwatch:PutMetricData"]
    resources = ["*"]
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

data "aws_iam_policy_document" "fleet-execution" {
  // allow fleet application to obtain the database password from secrets manager
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [var.fleet_config.database.password_secret_arn]
  }
}

resource "aws_iam_role" "main" {
  count              = var.fleet_config.iam_role_arn == null ? 1 : 0
  name               = var.fleet_config.iam.role.name
  description        = "IAM role that Fleet application assumes when running in ECS"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_policy" "main" {
  count       = var.fleet_config.iam_role_arn == null ? 1 : 0
  name        = var.fleet_config.iam.role.policy_name
  description = "IAM policy that Fleet application uses to define access to AWS resources"
  policy      = data.aws_iam_policy_document.fleet.json
}

resource "aws_iam_role_policy_attachment" "main" {
  count      = var.fleet_config.iam_role_arn == null ? 1 : 0
  policy_arn = aws_iam_policy.main[0].arn
  role       = aws_iam_role.main[0].name
}

resource "aws_iam_role_policy_attachment" "extras" {
  for_each   = toset(var.fleet_config.extra_iam_policies)
  policy_arn = each.value
  role       = aws_iam_role.main[0].name
}

resource "aws_iam_role_policy_attachment" "execution_extras" {
  for_each   = toset(var.fleet_config.extra_execution_iam_policies)
  policy_arn = each.value
  role       = aws_iam_role.execution.name
}

resource "aws_iam_policy" "execution" {
  name        = var.fleet_config.iam.execution.policy_name
  description = "IAM policy that Fleet application uses to define access to AWS resources"
  policy      = data.aws_iam_policy_document.fleet-execution.json
}

resource "aws_iam_role_policy_attachment" "execution" {
  policy_arn = aws_iam_policy.execution.arn
  role       = aws_iam_role.execution.name
}

resource "aws_iam_role" "execution" {
  name               = var.fleet_config.iam.execution.name
  description        = "The execution role for Fleet in ECS"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "role_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.execution.name
}

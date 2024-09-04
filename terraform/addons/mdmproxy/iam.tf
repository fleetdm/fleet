data "aws_iam_policy_document" "mdmproxy" {
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

data "aws_iam_policy_document" "mdm_execution" {
  // allow fleet application to obtain the database password from secrets manager
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.mdmproxy.arn]
  }
}

resource "aws_iam_role_policy_attachment" "execution_extras" {
  for_each   = toset(var.config.extra_execution_iam_policies)
  policy_arn = each.value
  role       = aws_iam_role.execution.name
}

resource "aws_iam_policy" "execution" {
  name        = var.config.iam.execution.policy_name
  description = "IAM policy that Fleet mdmproxy uses to define access to AWS resources"
  policy      = data.aws_iam_policy_document.mdm_execution.json
}

resource "aws_iam_role_policy_attachment" "execution" {
  policy_arn = aws_iam_policy.execution.arn
  role       = aws_iam_role.execution.name
}

resource "aws_iam_role" "execution" {
  name               = var.config.iam.execution.name
  description        = "The execution role for the mdmproxy in ECS"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "role_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.execution.name
}

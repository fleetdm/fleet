data "aws_iam_policy_document" "sfn-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "sfn" {
  role       = aws_iam_role.sfn.id
  policy_arn = aws_iam_policy.sfn.arn
}

resource "aws_iam_policy" "sfn" {
  name   = "${local.full_name}-sfn"
  policy = data.aws_iam_policy_document.sfn.json
}

data "aws_iam_policy_document" "sfn" {
  statement {
    actions   = ["ecs:RunTask"]
    resources = [replace(aws_ecs_task_definition.deprovisioner.arn, "/:\\d+$/", ":*"), replace(aws_ecs_task_definition.deprovisioner.arn, "/:\\d+$/", "")]
    condition {
      test     = "ArnLike"
      variable = "ecs:cluster"
      values   = [var.ecs_cluster.arn]
    }
  }
  statement {
    actions   = ["iam:PassRole"]
    resources = ["*"]
    condition {
      test     = "StringLike"
      variable = "iam:PassedToService"
      values   = ["ecs-tasks.amazonaws.com"]
    }
  }
  statement {
    actions   = ["events:PutTargets", "events:PutRule", "events:DescribeRule"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "sfn" {
  name = "${local.full_name}-sfn"

  assume_role_policy = data.aws_iam_policy_document.sfn-assume-role.json
}

resource "aws_iam_role_policy_attachment" "deprovisioner" {
  role       = aws_iam_role.deprovisioner.id
  policy_arn = aws_iam_policy.deprovisioner.arn
}

resource "aws_iam_policy" "deprovisioner" {
  name   = "${local.full_name}-deprovisioner"
  policy = data.aws_iam_policy_document.deprovisioner.json
}

data "aws_iam_policy_document" "deprovisioner" {
  statement {
    actions = [
      "dynamodb:List*",
      "dynamodb:DescribeReservedCapacity*",
      "dynamodb:DescribeLimits",
      "dynamodb:DescribeTimeToLive"
    ]
    resources = ["*"]
  }

  statement {
    actions = [
      "dynamodb:BatchGet*",
      "dynamodb:DescribeStream",
      "dynamodb:DescribeTable",
      "dynamodb:Get*",
      "dynamodb:Query",
      "dynamodb:Scan",
      "dynamodb:BatchWrite*",
      "dynamodb:CreateTable",
      "dynamodb:Delete*",
      "dynamodb:Update*",
      "dynamodb:PutItem"
    ]
    resources = [var.dynamodb_table.arn]
  }

  statement {
    actions = [ #tfsec:ignore:aws-iam-no-policy-wildcards
      "kms:Encrypt*",
      "kms:Decrypt*",
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Describe*"
    ]
    resources = [aws_kms_key.ecr.arn, var.kms_key.arn]
  }

  statement {
    actions   = ["*"]
    resources = ["*"]
  }

  statement {
    actions = ["eks:DescribeCluster"]
    resources = [var.eks_cluster.arn]
  }

  statement {
    actions = ["sts:GetCallerIdentity"]
    resources = ["*"]
  }
}

data "aws_iam_policy_document" "deprovisioner-assume-role" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "deprovisioner" {
  name = "${local.full_name}-deprovisioner"

  assume_role_policy = data.aws_iam_policy_document.deprovisioner-assume-role.json
}

resource "aws_ecs_task_definition" "deprovisioner" {
  family                   = "${local.full_name}-deprovisioner"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.deprovisioner.arn
  task_role_arn            = aws_iam_role.deprovisioner.arn
  cpu                      = 1024
  memory                   = 4096
  container_definitions = jsonencode(
    [
      {
        name        = "${var.prefix}-deprovisioner"
        image       = docker_registry_image.deprovisioner.name
        mountPoints = []
        volumesFrom = []
        essential   = true
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.main.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "${local.full_name}-deprovisioner"
          }
        },
        environment = concat([
          {
            name  = "TF_VAR_mysql_secret"
            value = var.mysql_secret.id
          },
          {
            name  = "TF_VAR_eks_cluster"
            value = var.eks_cluster.eks_cluster_id
          },
        ])
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_ecs_task_definition" "ingress_destroyer" {
  family                   = "${local.full_name}-ingress-destroyer"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.deprovisioner.arn
  task_role_arn            = aws_iam_role.deprovisioner.arn
  cpu                      = 512
  memory                   = 1024
  container_definitions = jsonencode(
    [
      {
        name        = "${var.prefix}-ingress-destroyer"
        image       = docker_registry_image.ingress_destroyer.name
        mountPoints = []
        volumesFrom = []
        essential   = true
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.main.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "${local.full_name}-ingress-destroyer"
          }
        },
        environment = [
          {
            name  = "CLUSTER_NAME"
            value = var.eks_cluster.eks_cluster_id
          },
        ]
      }
    ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group" "deprovisioner" {
  name        = "${local.full_name}-deprovisioner"
  description = "security group for ${local.full_name}-deprovisioner"
  vpc_id      = var.vpc.vpc_id

  egress {
    description      = "egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}


resource "aws_sfn_state_machine" "main" {
  name     = var.prefix
  role_arn = aws_iam_role.sfn.arn

  definition = <<EOF
{
  "Comment": "Controls the lifecycle of a Fleet demo environment",
  "StartAt": "Wait",
  "States": {
    "Wait": {
      "Type": "Wait",
      "SecondsPath": "$.waitTime",
      "Next": "IngressDestroyer"
    },
    "IngressDestroyer": {
      "Type": "Task",
      "Resource": "arn:aws:states:::ecs:runTask.sync",
      "Parameters": {
        "LaunchType": "FARGATE",
        "NetworkConfiguration": {
          "AwsvpcConfiguration": {
            "Subnets": ${jsonencode(var.vpc.private_subnets)},
            "SecurityGroups": ["${aws_security_group.deprovisioner.id}"],
            "AssignPublicIp": "DISABLED"
          }
        },
        "Cluster": "${var.ecs_cluster.arn}",
        "TaskDefinition": "${replace(aws_ecs_task_definition.deprovisioner.arn, "/:\\d+$/", "")}",
        "Overrides": {
          "ContainerOverrides": [
            {
              "Name": "${var.prefix}-ingress-destroyer",
              "Environment": [
                {
                  "Name": "INSTANCE_ID",
                  "Value.$": "$.instanceID"
                }
              ]
            }
          ]
        }
      },
      "Next": "Idle"
    },
    "Idle": {
      "Type": "Wait",
      "SecondsPath": "$.waitTime",
      "Next": "Deprovisioner"
    },
    "Deprovisioner": {
      "Type": "Task",
      "Resource": "arn:aws:states:::ecs:runTask.sync",
      "Parameters": {
        "LaunchType": "FARGATE",
        "NetworkConfiguration": {
          "AwsvpcConfiguration": {
            "Subnets": ${jsonencode(var.vpc.private_subnets)},
            "SecurityGroups": ["${aws_security_group.deprovisioner.id}"],
            "AssignPublicIp": "DISABLED"
          }
        },
        "Cluster": "${var.ecs_cluster.arn}",
        "TaskDefinition": "${replace(aws_ecs_task_definition.deprovisioner.arn, "/:\\d+$/", "")}",
        "Overrides": {
          "ContainerOverrides": [
            {
              "Name": "${var.prefix}-deprovisioner",
              "Environment": [
                {
                  "Name": "INSTANCE_ID",
                  "Value.$": "$.instanceID"
                }
              ]
            }
          ]
        }
      },
      "End": true
    }
  }
}
EOF
}

output "deprovisioner" {
  value = aws_sfn_state_machine.main
}

resource "random_uuid" "deprovisioner" {
  keepers = {
    lambda = data.archive_file.deprovisioner.output_sha
  }
}

resource "random_uuid" "ingress-destroyer" {
  keepers = {
    lambda = data.archive_file.ingress_destroyer.output_sha
  }
}

resource "local_file" "backend-config" {
  content = templatefile("${path.module}/deprovisioner/backend-template.conf",
    {
      remote_state = var.remote_state
  })
  filename = "${path.module}/deprovisioner/deploy_terraform/backend.conf"
}

data "archive_file" "deprovisioner" {
  type        = "zip"
  output_path = "${path.module}/.deprovisioner.zip"
  source_dir  = "${path.module}/deprovisioner"
}

// used to obtain a unique checksum
data "archive_file" "ingress_destroyer" {
  type        = "zip"
  output_path = "${path.module}/.ingress_destroyer.zip"
  source_dir  = "${path.module}/ingress_destroyer"
}

resource "docker_registry_image" "deprovisioner" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.deprovisioner.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/deprovisioner/"
    pull_parent = true
    platform    = "linux/amd64"
  }

  depends_on = [
    local_file.backend-config
  ]
}

resource "docker_registry_image" "ingress_destroyer" {
  name          = "${aws_ecr_repository.main.repository_url}:${data.git_repository.main.branch}-${random_uuid.ingress-destroyer.result}"
  keep_remotely = true

  build {
    context     = "${path.module}/ingress_destroyer/"
    pull_parent = true
    platform    = "linux/amd64"
  }

  depends_on = [
    local_file.backend-config
  ]
}


output "deprovisioner_role" {
  value = aws_iam_role.deprovisioner
}

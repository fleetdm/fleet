terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.6.2"
    }
  }

  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/android-amapi-mock/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "loadtest-android-mock"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform/android_amapi_mock"
      workspace   = terraform.workspace
    }
  }
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

provider "docker" {
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

data "aws_ecr_authorization_token" "token" {}

# Read shared VPC from remote state
data "terraform_remote_state" "shared" {
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/shared/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

# Read infra state for ECS cluster, IAM roles, ALB
data "terraform_remote_state" "infra" {
  backend   = "s3"
  workspace = terraform.workspace
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
}

locals {
  customer = "fleet-${terraform.workspace}"
}

# ---- ECR + Docker image ----

resource "aws_ecr_repository" "android_amapi_mock" {
  name                 = "${local.customer}-android-mock"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  force_delete = true
}

resource "docker_image" "android_amapi_mock" {
  name = "${aws_ecr_repository.android_amapi_mock.repository_url}:${var.tag}"

  build {
    context    = "${path.module}/../docker/"
    dockerfile = "android-amapi-mock.Dockerfile"
    platform   = "linux/amd64"
    build_args = {
      TAG = var.tag
    }
  }
}

resource "docker_tag" "android_amapi_mock" {
  source_image = docker_image.android_amapi_mock.name
  target_image = "${aws_ecr_repository.android_amapi_mock.repository_url}:${var.tag}"
}

resource "docker_registry_image" "android_amapi_mock" {
  name          = docker_tag.android_amapi_mock.target_image
  keep_remotely = true
}

# ---- CloudWatch Logs ----

resource "aws_cloudwatch_log_group" "android_amapi_mock" {
  name              = "${local.customer}-android-mock"
  retention_in_days = 30
}

# ---- Security Group ----

resource "aws_security_group" "android_amapi_mock" {
  name_prefix = "${local.customer}-android-mock-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Android AMAPI mock - allows HTTP from internal ALB"

  ingress {
    description     = "HTTP from internal ALB"
    from_port       = 9999
    to_port         = 9999
    protocol        = "tcp"
    security_groups = [data.terraform_remote_state.infra.outputs.internal_alb_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ---- IAM: allow the execution role to read the Google credentials secret ----

resource "aws_iam_role_policy" "android_mock_secrets" {
  count = var.enable_google_forwarding ? 1 : 0
  name  = "${local.customer}-android-mock-secrets"
  role  = basename(data.terraform_remote_state.infra.outputs.ecs_execution_arn)
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = [data.terraform_remote_state.shared.outputs.android_google_credentials.arn]
      }
    ]
  })
}

# ---- ECS Task Definition ----

resource "aws_ecs_task_definition" "android_amapi_mock" {
  family                   = "${local.customer}-android-mock"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256
  memory                   = 512
  execution_role_arn       = data.terraform_remote_state.infra.outputs.ecs_execution_arn
  task_role_arn            = data.terraform_remote_state.infra.outputs.ecs_arn

  container_definitions = jsonencode([
    {
      name      = "android-amapi-mock"
      image     = docker_registry_image.android_amapi_mock.name
      essential = true

      portMappings = [
        {
          containerPort = 9999
          protocol      = "tcp"
        }
      ]

      command = ["--listen", ":9999"]

      environment = []

      secrets = var.enable_google_forwarding ? [
        {
          name      = "GOOGLE_CREDENTIALS"
          valueFrom = data.terraform_remote_state.shared.outputs.android_google_credentials.arn
        }
      ] : []

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.android_amapi_mock.name
          "awslogs-region"        = data.aws_region.current.name
          "awslogs-stream-prefix" = "android-mock"
        }
      }
    }
  ])
}

# ---- ECS Service ----

resource "aws_ecs_service" "android_amapi_mock" {
  name            = "${local.customer}-android-mock"
  cluster         = data.terraform_remote_state.infra.outputs.ecs_cluster
  task_definition = aws_ecs_task_definition.android_amapi_mock.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = data.terraform_remote_state.infra.outputs.vpc_subnets
    security_groups = [aws_security_group.android_amapi_mock.id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.android_amapi_mock.arn
    container_name   = "android-amapi-mock"
    container_port   = 9999
  }

  depends_on = [
    aws_lb_listener_rule.android_amapi_mock_v1,
    aws_lb_listener_rule.android_amapi_mock_coordination,
    aws_iam_role_policy.android_mock_secrets,
  ]
}

# ---- ALB target group + listener rules ----

resource "aws_lb_target_group" "android_amapi_mock" {
  name                 = "${local.customer}-android-mock"
  protocol             = "HTTP"
  port                 = 9999
  target_type          = "ip"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 10

  health_check {
    path                = "/mock/health"
    matcher             = "200"
    timeout             = 5
    interval            = 30
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

# Route AMAPI requests (/v1/*) to the mock
resource "aws_lb_listener_rule" "android_amapi_mock_v1" {
  listener_arn = data.terraform_remote_state.infra.outputs.internal_alb_listener_arn
  priority     = 20

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.android_amapi_mock.arn
  }

  condition {
    path_pattern {
      values = ["/v1/*"]
    }
  }
}

# Route coordination API requests (/mock/*) to the mock
resource "aws_lb_listener_rule" "android_amapi_mock_coordination" {
  listener_arn = data.terraform_remote_state.infra.outputs.internal_alb_listener_arn
  priority     = 21

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.android_amapi_mock.arn
  }

  condition {
    path_pattern {
      values = ["/mock/*"]
    }
  }
}

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
    random = {
      source  = "hashicorp/random"
      version = ">= 3.6.0"
    }
  }

  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/activity-driver/terraform.tfstate"
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
      environment = "loadtest-activity-driver"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform/activity_driver"
      workspace   = terraform.workspace
    }
  }
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}
data "aws_ecr_authorization_token" "token" {}

provider "docker" {
  registry_auth {
    address  = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com"
    username = data.aws_ecr_authorization_token.token.user_name
    password = data.aws_ecr_authorization_token.token.password
  }
}

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
  # The load-test terraform names the ECS cluster, the Aurora cluster and the Fleet log group after the instance.
  customer  = "fleet-${terraform.workspace}"
  name      = "${local.customer}-activity-driver"
  image_tag = replace(var.git_ref, "/", "-")
  fleet_url = var.fleet_url != "" ? var.fleet_url : "http://${data.terraform_remote_state.infra.outputs.internal_alb_dns_name}"
}

# ---- Image: the driver and fleetctl, built from git_ref ----

resource "aws_ecr_repository" "driver" {
  name                 = local.name
  image_tag_mutability = "MUTABLE"
  force_delete         = true

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "docker_image" "driver" {
  name = "${aws_ecr_repository.driver.repository_url}:${local.image_tag}"

  build {
    context    = "${path.module}/../docker/"
    dockerfile = "activity-driver.Dockerfile"
    platform   = "linux/amd64"
    build_args = {
      TAG = var.git_ref
    }
  }
}

resource "docker_registry_image" "driver" {
  name          = docker_image.driver.name
  keep_remotely = true
}

# ---- Bucket: the stop marker and the private GitOps bundle ----

resource "aws_s3_bucket" "driver" {
  bucket_prefix = "${local.name}-"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "driver" {
  bucket                  = aws_s3_bucket.driver.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "driver" {
  bucket = aws_s3_bucket.driver.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# ---- Secrets: values for token, profile and notify are set by hand (README), never in terraform ----

resource "aws_secretsmanager_secret" "token" {
  name_prefix             = "${local.name}-token-"
  description             = "Fleet API token of an admin API-only user the activity driver acts as"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret" "profile" {
  count                   = var.with_profile ? 1 : 0
  name_prefix             = "${local.name}-profile-"
  description             = "Private customer profile YAML (measured activity rates) for the activity driver"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret" "notify" {
  count                   = var.notify ? 1 : 0
  name_prefix             = "${local.name}-notify-"
  description             = "Webhook URL the activity driver posts trips to"
  recovery_window_in_days = 0
}

# The driver's password users get passwords derived from this, so nothing has to survive a task restart.
resource "random_password" "users" {
  length  = 48
  special = false
}

resource "aws_secretsmanager_secret" "users" {
  name_prefix             = "${local.name}-users-"
  description             = "Secret the activity driver derives its password users' passwords from"
  recovery_window_in_days = 0
}

resource "aws_secretsmanager_secret_version" "users" {
  secret_id     = aws_secretsmanager_secret.users.id
  secret_string = random_password.users.result
}

# ---- Logs ----

resource "aws_cloudwatch_log_group" "driver" {
  name              = local.name
  retention_in_days = 90
}

# ---- IAM: its own roles, so the shared Fleet roles stay as they are ----

data "aws_iam_policy_document" "ecs_tasks_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "execution" {
  name_prefix        = "lt-driver-exec-"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume.json
}

resource "aws_iam_role_policy_attachment" "execution" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "execution_secrets" {
  name = "secrets"
  role = aws_iam_role.execution.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = ["secretsmanager:GetSecretValue"]
      Resource = concat(
        [aws_secretsmanager_secret.token.arn, aws_secretsmanager_secret.users.arn],
        aws_secretsmanager_secret.profile[*].arn,
        aws_secretsmanager_secret.notify[*].arn,
      )
    }]
  })
}

# The task reads the instance (checks) and its own bucket: nothing it can change in Fleet's infrastructure.
resource "aws_iam_role" "task" {
  name_prefix        = "lt-driver-task-"
  assume_role_policy = data.aws_iam_policy_document.ecs_tasks_assume.json
}

resource "aws_iam_role_policy" "task" {
  name = "read-instance"
  role = aws_iam_role.task.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "FleetService"
        Effect   = "Allow"
        Action   = ["ecs:DescribeServices", "ecs:ListTasks", "ecs:DescribeTasks"]
        Resource = "*"
      },
      {
        Sid      = "Metrics"
        Effect   = "Allow"
        Action   = ["cloudwatch:GetMetricData", "cloudwatch:ListMetrics"]
        Resource = "*"
      },
      {
        Sid      = "FleetLogs"
        Effect   = "Allow"
        Action   = ["logs:StartQuery"]
        Resource = "arn:aws:logs:${data.aws_region.current.name}:${data.aws_caller_identity.current.account_id}:log-group:${local.customer}:*"
      },
      {
        Sid      = "LogsQueryResults"
        Effect   = "Allow"
        Action   = ["logs:GetQueryResults", "logs:StopQuery"]
        Resource = "*"
      },
      {
        Sid      = "StopMarker"
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject"]
        Resource = "${aws_s3_bucket.driver.arn}/STOP"
      },
      {
        Sid      = "GitopsBundle"
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = "${aws_s3_bucket.driver.arn}/gitops/*"
      },
      {
        # GetObject on a missing key answers 403 instead of 404 without ListBucket, and the marker is usually missing.
        Sid      = "StopMarkerMissing"
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = aws_s3_bucket.driver.arn
      },
    ]
  })
}

# ---- Network: outbound only (Fleet's load balancer, AWS APIs) ----

resource "aws_security_group" "driver" {
  name_prefix = "${local.name}-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Load-test activity driver: outbound only"

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

# ---- Task and service ----

resource "aws_ecs_task_definition" "driver" {
  family                   = local.name
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.task_size.cpu
  memory                   = var.task_size.memory
  execution_role_arn       = aws_iam_role.execution.arn
  task_role_arn            = aws_iam_role.task.arn

  container_definitions = jsonencode([{
    name      = "driver"
    image     = docker_registry_image.driver.name
    essential = true

    command = concat(
      [
        "-lt", local.customer,
        "-aws_region", data.aws_region.current.name,
        "-stop_file", "s3://${aws_s3_bucket.driver.id}/STOP",
        "-users_secret_env", "DRIVER_USERS_SECRET",
      ],
      var.notify ? ["-notify_url_env", "DRIVER_NOTIFY_URL"] : [],
      var.extra_flags,
    )

    environment = concat(
      [
        { name = "FLEET_URL", value = local.fleet_url },
        { name = "AWS_REGION", value = data.aws_region.current.name },
      ],
      var.gitops_bundle_key != "" ? [{ name = "DRIVER_GITOPS_BUNDLE", value = "s3://${aws_s3_bucket.driver.id}/gitops/${var.gitops_bundle_key}" }] : [],
    )

    secrets = concat(
      [
        { name = "FLEET_DRIVER_TOKEN", valueFrom = aws_secretsmanager_secret.token.arn },
        { name = "DRIVER_USERS_SECRET", valueFrom = aws_secretsmanager_secret.users.arn },
      ],
      [for s in aws_secretsmanager_secret.profile : { name = "DRIVER_PROFILE", valueFrom = s.arn }],
      [for s in aws_secretsmanager_secret.notify : { name = "DRIVER_NOTIFY_URL", valueFrom = s.arn }],
    )

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.driver.name
        "awslogs-region"        = data.aws_region.current.name
        "awslogs-stream-prefix" = "driver"
      }
    }
  }])
}

resource "aws_ecs_service" "driver" {
  name            = local.name
  cluster         = data.terraform_remote_state.infra.outputs.ecs_cluster
  task_definition = aws_ecs_task_definition.driver.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  # One driver at a time: stop the old task before starting the new one on a deploy.
  deployment_minimum_healthy_percent = 0
  deployment_maximum_percent         = 100

  network_configuration {
    subnets         = data.terraform_remote_state.infra.outputs.vpc_subnets
    security_groups = [aws_security_group.driver.id]
  }

  depends_on = [aws_iam_role_policy.execution_secrets]
}

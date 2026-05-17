terraform {
  required_version = ">= 1.5"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.68.0"
    }
  }

  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/pmm/terraform.tfstate"
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
  region = var.aws_region
  default_tags {
    tags = {
      environment = "loadtest-pmm"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform/pmm"
      workspace   = terraform.workspace
    }
  }
}

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
  workspace = terraform.workspace
}

# Read infra state for ECS cluster, IAM roles, RDS info
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

# --- PMM Admin Password ---

resource "random_password" "pmm_admin" {
  length  = 24
  special = false
}

resource "aws_secretsmanager_secret" "pmm_admin_password" {
  name                    = "${local.customer}-pmm-admin-password"
  recovery_window_in_days = 0 # Ephemeral loadtest, allow immediate deletion
}

resource "aws_secretsmanager_secret_version" "pmm_admin_password" {
  secret_id     = aws_secretsmanager_secret.pmm_admin_password.id
  secret_string = random_password.pmm_admin.result
}

data "aws_secretsmanager_secret" "rds_password" {
  name = "${local.customer}-database-password"
}

resource "aws_iam_role" "pmm_execution" {
  name = "${local.customer}-pmm-execution"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

resource "aws_iam_role_policy" "pmm_execution" {
  name = "${local.customer}-pmm-execution"
  role = aws_iam_role.pmm_execution.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = "${aws_cloudwatch_log_group.pmm.arn}:*"
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
        ]
        Resource = [
          aws_secretsmanager_secret.pmm_admin_password.arn,
          data.aws_secretsmanager_secret.rds_password.arn,
        ]
      },
    ]
  })
}

# --- DNS zone ---

data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}

# --- CloudWatch Logs ---

resource "aws_cloudwatch_log_group" "pmm" {
  name              = "${local.customer}-pmm"
  retention_in_days = 30
}

# --- ECS Task Definition ---

resource "aws_ecs_task_definition" "pmm" {
  family                   = "${local.customer}-pmm"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 2048
  memory                   = 4096
  execution_role_arn       = aws_iam_role.pmm_execution.arn
  task_role_arn            = data.terraform_remote_state.infra.outputs.ecs_arn

  ephemeral_storage {
    size_in_gib = 100
  }

  container_definitions = jsonencode([
    {
      name      = "pmm-server"
      image     = "percona/pmm-server@sha256:4dd6dab43e7b11b9bc9c6284b690e8056210a047a93793fceb5a0ab706ac3fe4" # percona/pmm-server:2
      essential = true

      portMappings = [
        {
          containerPort = 443
          protocol      = "tcp"
        }
      ]

      entryPoint = ["/bin/bash", "-c"]
      command = [join("\n", [
        "# Background: wait for PMM server, then configure it",
        "(",
        "  echo 'Waiting for PMM server to become ready...'",
        "  until curl -sSf -k https://localhost:443/v1/readyz > /dev/null 2>&1; do",
        "    sleep 5",
        "  done",
        "  echo 'PMM server is ready'",
        "",
        "  # Change default admin password",
        "  curl -sSf -k -X PATCH https://localhost:443/v1/users \\",
        "    -H 'Content-Type: application/json' \\",
        "    -u admin:admin \\",
        "    -d \"{\\\"new_password\\\": \\\"$PMM_ADMIN_PASSWORD\\\"}\"",
        "  echo 'Admin password changed'",
        "",
        "  # Add RDS MySQL monitoring",
        "  pmm-admin add mysql \\",
        "    --server-url=\"https://admin:$PMM_ADMIN_PASSWORD@localhost:443\" \\",
        "    --server-insecure-tls \\",
        "    --username=\"$PMM_MYSQL_USERNAME\" \\",
        "    --password=\"$PMM_MYSQL_PASSWORD\" \\",
        "    --host=\"$PMM_MYSQL_HOST\" \\",
        "    --port=3306 \\",
        "    --query-source=perfschema \\",
        "    fleet-mysql",
        "  echo 'MySQL monitoring configured'",
        ") &",
        "",
        "# Run PMM server as PID 1 (foreground)",
        "exec /opt/entrypoint.sh",
      ])]

      environment = [
        {
          name  = "PMM_MYSQL_HOST"
          value = data.terraform_remote_state.infra.outputs.rds_cluster_endpoint
        },
        {
          name  = "PMM_MYSQL_USERNAME"
          value = data.terraform_remote_state.infra.outputs.rds_cluster_master_username
        },
      ]

      secrets = [
        {
          name      = "PMM_ADMIN_PASSWORD"
          valueFrom = aws_secretsmanager_secret.pmm_admin_password.arn
        },
        {
          name      = "PMM_MYSQL_PASSWORD"
          valueFrom = "${data.aws_secretsmanager_secret.rds_password.arn}:password::"
        },
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.pmm.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "pmm"
        }
      }
    }
  ])
}

# --- ECS Service ---

resource "aws_ecs_service" "pmm" {
  name            = "${local.customer}-pmm"
  cluster         = data.terraform_remote_state.infra.outputs.ecs_cluster
  task_definition = aws_ecs_task_definition.pmm.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = data.terraform_remote_state.infra.outputs.vpc_subnets
    security_groups = data.terraform_remote_state.infra.outputs.security_groups
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.pmm.arn
    container_name   = "pmm-server"
    container_port   = 443
  }
}

# --- Internal ALB target group + listener rule ---

resource "aws_lb_target_group" "pmm" {
  name                 = "${local.customer}-pmm"
  protocol             = "HTTPS"
  port                 = 443
  target_type          = "ip"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 30

  health_check {
    protocol            = "HTTPS"
    path                = "/v1/readyz"
    matcher             = "200"
    timeout             = 10
    interval            = 30
    healthy_threshold   = 3
    unhealthy_threshold = 5
  }
}

resource "aws_lb_listener_rule" "pmm" {
  listener_arn = data.terraform_remote_state.infra.outputs.internal_alb_listener_arn
  priority     = 10

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.pmm.arn
  }

  condition {
    host_header {
      values = ["pmm.${terraform.workspace}.loadtest.fleetdm.com"]
    }
  }
}

# --- DNS ---

resource "aws_route53_record" "pmm" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = "pmm.${terraform.workspace}.loadtest.fleetdm.com"
  type    = "A"

  alias {
    name                   = data.terraform_remote_state.infra.outputs.internal_alb_dns_name
    zone_id                = data.terraform_remote_state.infra.outputs.internal_alb_zone_id
    evaluate_target_health = true
  }
}

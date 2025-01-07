provider "aws" {
  region = "us-east-2"
  default_tags {
    tags = {
      environment = "elastic-agent"
      terraform   = "https://github.com/fleetdm/fleet/tree/main/infrastructure/infrastructure/elastic-agent"
      state       = "s3://fleet-terraform-state20220408141538466600000002/infrastructure/elastic-agent/terraform.tfstate"
    }
  }
}

data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

variable "fleet_url" {}
variable "fleet_enroll_token" {}
variable "kibana_fleet_password" {}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.61.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "infrastructure/elastic-agent/terraform.tfstate" # This should be set to account_alias/unique_key/terraform.tfstate
    workspace_key_prefix = "infrastructure"                                 # This should be set to the account alias
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
  }
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.12.0"

  name = "elastic-agent"
  cidr = "10.10.0.0/16"

  azs = ["us-east-2a", "us-east-2b", "us-east-2c"]
  private_subnets = [
    "10.10.16.0/20",
    "10.10.32.0/20",
    "10.10.48.0/20",
  ]
  public_subnets = [
    "10.10.128.0/24",
    "10.10.129.0/24",
    "10.10.130.0/24",
  ]

  create_database_subnet_group       = false
  create_database_subnet_route_table = false

  create_elasticache_subnet_group       = false
  create_elasticache_subnet_route_table = false

  enable_vpn_gateway     = false
  one_nat_gateway_per_az = false

  single_nat_gateway = true
  enable_nat_gateway = true
}

resource "aws_ecs_cluster" "main" {
  name = "main"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

output "ecs_cluster" {
  value = aws_ecs_cluster.main
}

resource "aws_ecs_service" "main" {
  name                               = "elastic-agent"
  launch_type                        = "FARGATE"
  cluster                            = aws_ecs_cluster.main.id
  task_definition                    = aws_ecs_task_definition.main.arn
  desired_count                      = 1
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = module.vpc.private_subnets
    security_groups = [aws_security_group.main.id]
  }
}

resource "aws_ecs_task_definition" "main" {
  family                   = "elastic-agent"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.execution.arn
  cpu                      = 256
  memory                   = 512
  container_definitions = jsonencode(
    [
      {
        name        = "elastic-agent"
        image       = "docker.elastic.co/beats/elastic-agent:8.7.0"
        cpu         = 256
        memory      = 512
        essential   = true
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = {
            awslogs-group         = aws_cloudwatch_log_group.main.name
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "elastic-agent"
          }
        },
        environment = [
          {
            name  = "FLEET_ENROLL"
            value = "1"
          },
          {
            name  = "FLEET_URL"
            value = var.fleet_url
          },
          {
            name  = "FLEET_ENROLLMENT_TOKEN"
            value = var.fleet_enroll_token
          },
          {
            name  = "KIBANA_HOST"
            value = "http://kibana:5601"
          },
          {
            name  = "KIBANA_FLEET_USERNAME"
            value = "elastic"
          },
          {
            name  = "KIBANA_FLEET_PASSWORD"
            value = var.kibana_fleet_password
          },
        ]
      }
  ])
}

resource "aws_cloudwatch_log_group" "main" {
  name              = "elastic-agent"
  retention_in_days = 30
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

resource "aws_iam_role" "execution" {
  name               = "elastic-agent"
  description        = "The execution role for Elastic Agent"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_iam_role_policy_attachment" "role_attachment" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
  role       = aws_iam_role.execution.name
}

resource "aws_security_group" "main" {
  name        = "elastic-agent"
  description = "Elastic Agent Service Security Group"
  vpc_id      = module.vpc.vpc_id
  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

data "aws_region" "current" {}

locals {
  mdmproxy_secrets = {
    MDMPROXY_AUTH_TOKEN    = var.config.auth_token
    MDMPROXY_MIGRATE_UDIDS = join(" ", var.config.migrate_udids)
  }
}

resource "aws_security_group" "mdmproxy" {
  count       = var.config.networking.security_groups == null ? 1 : 0
  name        = var.config.networking.security_group_name
  description = "Fleet ECS Service Security Group"
  vpc_id      = var.vpc_id
  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  ingress {
    description      = "Ingress only on container port"
    from_port        = 8080
    to_port          = 8080
    protocol         = "TCP"
    cidr_blocks      = var.config.networking.ingress_sources.cidr_blocks
    ipv6_cidr_blocks = var.config.networking.ingress_sources.ipv6_cidr_blocks
    security_groups  = concat(var.config.networking.ingress_sources.security_groups, [aws_security_group.alb.id])
    prefix_list_ids  = var.config.networking.ingress_sources.prefix_list_ids
  }
}

module "alb" {
  source  = "terraform-aws-modules/alb/aws"
  version = "8.3.0"

  name = var.alb_config.name

  load_balancer_type = "application"

  vpc_id          = var.vpc_id
  subnets         = var.alb_config.subnets
  security_groups = concat(var.alb_config.security_groups, [aws_security_group.alb.id])
  access_logs     = var.alb_config.access_logs
  idle_timeout    = var.alb_config.idle_timeout

  target_groups = concat([
    {
      name             = var.alb_config.name
      backend_protocol = "HTTP"
      backend_port     = 80
      target_type      = "ip"
      health_check = {
        path                = "/healthz"
        matcher             = "200"
        timeout             = 10
        interval            = 15
        healthy_threshold   = 5
        unhealthy_threshold = 5
      }
    }
  ], var.alb_config.extra_target_groups)

  # Require TLS 1.2 as earlier versions are insecure
  listener_ssl_policy_default = var.alb_config.tls_policy

  https_listeners = [
    {
      port               = 443
      protocol           = "HTTPS"
      certificate_arn    = var.alb_config.certificate_arn
      target_group_index = 0
    }
  ]

  https_listener_rules = var.alb_config.https_listener_rules

  http_tcp_listeners = [
    {
      port        = 80
      protocol    = "HTTP"
      action_type = "redirect"
      redirect = {
        port        = "443"
        protocol    = "HTTPS"
        status_code = "HTTP_301"
      }
    }
  ]
}

resource "aws_security_group" "alb" {
  #checkov:skip=CKV2_AWS_5:False positive
  vpc_id      = var.vpc_id
  description = "Fleet-mdmproxy ALB Security Group"
  ingress {
    description      = "Ingress from all, its a public load balancer"
    from_port        = 443
    to_port          = 443
    protocol         = "tcp"
    cidr_blocks      = var.alb_config.allowed_cidrs
    ipv6_cidr_blocks = var.alb_config.allowed_ipv6_cidrs
  }

  ingress {
    description      = "For http to https redirect"
    from_port        = 80
    to_port          = 80
    protocol         = "tcp"
    cidr_blocks      = var.alb_config.allowed_cidrs
    ipv6_cidr_blocks = var.alb_config.allowed_ipv6_cidrs
  }

  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = var.alb_config.egress_cidrs
    ipv6_cidr_blocks = var.alb_config.egress_ipv6_cidrs
  }
}

resource "aws_secretsmanager_secret" "mdmproxy" {
  name = "${var.customer_prefix}-mdmproxy"
}

resource "aws_secretsmanager_secret_version" "mdmproxy" {
  secret_id     = aws_secretsmanager_secret.mdmproxy.id
  secret_string = jsonencode(local.mdmproxy_secrets)
}

resource "aws_ecs_service" "mdmproxy" {
  name                               = "${var.customer_prefix}-mdmproxy"
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.mdmproxy.arn
  desired_count                      = var.config.desired_count
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200
  force_new_deployment               = true

  triggers = {
    redeployment = md5(jsonencode(aws_secretsmanager_secret_version.mdmproxy.secret_string))
  }

  load_balancer {
    target_group_arn = module.alb.target_group_arns[0]
    container_name   = "mdmproxy"
    container_port   = 8080
  }

  lifecycle {
    ignore_changes = [desired_count]
  }

  network_configuration {
    subnets         = var.config.networking.subnets
    security_groups = var.config.networking.security_groups == null ? aws_security_group.mdmproxy.*.id : var.config.networking.security_groups
  }
}

resource "aws_ecs_task_definition" "mdmproxy" {
  family                   = "${var.customer_prefix}-mdmproxy"
  cpu                      = var.config.cpu
  memory                   = var.config.mem
  execution_role_arn       = aws_iam_role.execution.arn
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]

  container_definitions = jsonencode([
    {
      name        = "mdmproxy"
      image       = var.config.image
      cpu         = var.config.cpu
      memory      = var.config.mem
      essential   = true
      portMappings = [
        {
          # This port is the same that the contained application also uses
          containerPort = 8080
          protocol      = "tcp"
        }
      ]
      networkMode = "awsvpc"
      secrets = [
        {
          name      = "MDMPROXY_AUTH_TOKEN"
          valueFrom = "${aws_secretsmanager_secret.mdmproxy.arn}:MDMPROXY_AUTH_TOKEN::"
        },
        {
          name      = "MDMPROXY_MIGRATE_UDIDS"
          valueFrom = "${aws_secretsmanager_secret.mdmproxy.arn}:MDMPROXY_MIGRATE_UDIDS::"
        },
      ]
      repositoryCredentials = var.config.repository_credentials
      ulimits = [
        {
          name      = "nofile"
          softLimit = 999999
          hardLimit = 999999
        }
      ]
      environment = [
        {
          name  = "MDMPROXY_SERVER_ADDRESS"
          value = ":8080"
        },
        {
          name  = "MDMPROXY_MIGRATE_PERCENTAGE"
          value = tostring(var.config.migrate_percentage)
        },
        {
          name  = "MDMPROXY_EXISTING_HOSTNAME"
          value = var.config.existing_hostname
        },
        {
          name  = "MDMPROXY_EXISTING_URL"
          value = var.config.existing_url
        },
        {
          name  = "MDMPROXY_FLEET_URL"
          value = var.config.fleet_url
        },
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = var.awslogs_config.group
          awslogs-region        = var.awslogs_config.region == null ? data.aws_region.current.name : var.awslogs_config.region
          awslogs-stream-prefix = "${var.awslogs_config.prefix}-mdmproxy"
        }
      }
    }
  ])
}




data "aws_region" "current" {}

resource "aws_cloudwatch_log_group" "saml_auth_proxy" {
  name = "${var.customer_prefix}-saml-auth-proxy"
}

data "aws_iam_policy_document" "saml_auth_proxy" {
  // allow saml_auth_proxy to obtain certs from secrets manager
  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.saml_auth_proxy_cert.arn]
  }
}

resource "aws_iam_policy" "saml_auth_proxy" {
  name        = "${var.customer_prefix}-saml-auth-proxy"
  description = "IAM Policy to provide saml_auth_proxy access to secrets"
  policy      = data.aws_iam_policy_document.saml_auth_proxy.json
}

resource "aws_secretsmanager_secret" "saml_auth_proxy_cert" {
  name_prefix = "${var.customer_prefix}-saml-auth-proxy-cert"
}

resource "aws_security_group" "saml_auth_proxy_alb" {
  #checkov:skip=CKV2_AWS_5:False positive
  vpc_id      = var.vpc_id
  description = "Fleet ALB Security Group"

  ingress {
    description     = "Internal HTTP back to Fleet"
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.saml_auth_proxy_service.id]
  }

  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = []
  }
}

resource "aws_security_group" "saml_auth_proxy_service" {
  #checkov:skip=CKV2_AWS_5:False positive
  vpc_id      = var.vpc_id
  description = "Fleet ALB Security Group"

  ingress {
    description = "Internal HTTP back to Fleet"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }

  egress {
    description      = "Egress to all"
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = []
  }
}


module "saml_auth_proxy_alb" {
  source  = "terraform-aws-modules/alb/aws"
  version = "8.2.1"

  name = "${var.customer_prefix}-saml-auth-proxy"

  load_balancer_type = "application"

  vpc_id          = var.vpc_id
  subnets         = var.subnets
  security_groups = [aws_security_group.saml_auth_proxy_alb.id]
  # FIXME: Get this working eventually.
  access_logs = var.alb_access_logs

  internal = true
  target_groups = [
    {
      name             = "${var.customer_prefix}-saml-to-fleet"
      backend_protocol = "HTTP"
      backend_port     = 8080
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
  ]

  http_tcp_listeners = [
    {
      port               = 8080
      protocol           = "HTTP"
      target_group_index = 0
    }
  ]
}

resource "aws_ecs_task_definition" "saml_auth_proxy" {
  family                   = "${var.customer_prefix}-saml-auth-proxy"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = var.ecs_execution_iam_role_arn
  task_role_arn            = var.ecs_iam_role_arn
  cpu                      = 256
  memory                   = 1024
  container_definitions = jsonencode(
    [
      {
        name        = "${var.customer_prefix}-saml-auth-proxy"
        image       = var.saml_auth_proxy_image
        cpu         = 256
        memory      = 512
        mountPoints = []
        volumesFrom = []
        essential   = true
        ulimits = [
          {
            softLimit = 9999,
            hardLimit = 9999,
            name      = "nofile"
          }
        ]
        portMappings = [
          {
            # This port is the same that the contained application also uses
            containerPort = 8080
            protocol      = "tcp"
          }
        ]
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = var.logging_options != null ? var.logging_options : {
            awslogs-group         = "${var.customer_prefix}-saml-auth-proxy"
            awslogs-region        = data.aws_region.current.name
            awslogs-stream-prefix = "saml-auth-proxy"
          }
        }
        workingDirectory = "/go",
        secrets = [
          {
            name      = "SAML_PROXY_SP_CERT_BYTES"
            valueFrom = "${aws_secretsmanager_secret.saml_auth_proxy_cert.arn}:cert::"
          },
          {
            name      = "SAML_PROXY_SP_KEY_BYTES"
            valueFrom = "${aws_secretsmanager_secret.saml_auth_proxy_cert.arn}:key::"
          },
        ]
        environment = [
          {
            name  = "SAML_PROXY_SP_CERT_PATH"
            value = "/tmp/saml-auth-proxy/cert.pem"
          },
          {
            name  = "SAML_PROXY_SP_KEY_PATH"
            value = "/tmp/saml-auth-proxy/key.pem"
          },
          {
            name  = "SAML_PROXY_BACKEND_URL"
            value = "http://${module.saml_auth_proxy_alb.lb_dns_name}:8080/"
          },
          {
            name  = "SAML_PROXY_IDP_METADATA_URL"
            value = var.idp_metadata_url
          },
          {
            name  = "SAML_PROXY_BASE_URL"
            value = var.base_url
          },
          {
            name  = "SAML_PROXY_COOKIE_MAX_AGE"
            value = var.cookie_max_age
          },
          {
            name  = "SAML_PROXY_COOKIE_NAME"
            value = "saml_auth_proxy_token"
          },
          {
            name  = "SAML_PROXY_ALLOW_IDP_INITIATED"
            value = "true"
          },
        ]
        entryPoint = ["/bin/sh"],
        command    = ["-c", file("${path.module}/files/saml-auth-proxy.sh")]
      }
    ]
  )
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_ecs_service" "saml_auth_proxy" {
  name                               = "${var.customer_prefix}_saml_auth_proxy"
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.saml_auth_proxy.arn
  desired_count                      = var.proxy_containers
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  network_configuration {
    subnets         = var.subnets
    security_groups = [aws_security_group.saml_auth_proxy_service.id]
  }

  load_balancer {
    target_group_arn = var.alb_target_group_arn
    container_name   = "${var.customer_prefix}-saml-auth-proxy"
    container_port   = 8080
  }
}

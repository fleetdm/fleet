resource "aws_kms_key" "enroll_secret" {
  deletion_window_in_days = 10
  enable_key_rotation     = true
}

resource "aws_kms_alias" "enroll_secret" {
  name_prefix   = "alias/${var.customer_prefix}-enroll-secret-key"
  target_key_id = aws_kms_key.enroll_secret.key_id
}

resource "aws_secretsmanager_secret" "enroll_secret" {
  name_prefix = "${var.customer_prefix}-enroll-secret"
  kms_key_id  = aws_kms_key.enroll_secret.arn
}

data "aws_secretsmanager_secret_version" "enroll_secret" {
  secret_id = aws_secretsmanager_secret.enroll_secret.id
}

module "saml_auth_proxy_alb" {
  source  = "terraform-aws-modules/alb/aws"
  version = "8.2.1"

  name = var.alb_config.name

  load_balancer_type = "application"

  vpc_id          = var.vpc_id
  subnets         = var.alb_config.subnets
  security_groups = concat(var.alb_config.security_groups, [aws_security_group.alb.id])
  access_logs     = var.alb_config.access_logs

  internal        = true
  target_groups = [
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
  ]

  # Require TLS 1.2 as earlier versions are insecure

  http_tcp_listeners = [
    {
      port        = 80
      protocol    = "HTTP"
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
        name        = "saml-auth-proxy"
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
        networkMode = "awsvpc"
        logConfiguration = {
          logDriver = "awslogs"
          options = var.logging_options
        }
        workingDirectory = "/go",
        secrets = [
          {
            name      = "SAML_PROXY_SP_CERT_BYTES"
            valueFrom = "${aws_secretsmanager_secret.saml_auth_proxy_cert.arn}:cert"
          },
          {
            name      = "SAML_PROXY_SP_KEY_BYTES"
            valueFrom = "${aws_secretsmanager_secret.saml_auth_proxy_cert.arn}:key"
          },
        ]
        environmnet = [
          {
            name  = "SAML_PROXY_SP_CERT_PATH"
            value = "/tmp/saml-auth-proxy/cert.pem
          },
          {
            name   = "SAML_PROXY_SP_KEY_PATH"
            value = "/tmp/saml-auth-proxy/key.pem"
          },
          {
            name   = "SAML_PROXY_SP_KEY_PATH"
            value = "/tmp/saml-auth-proxy/key.pem"
          },
          {
            name   = "SAML_PROXY_BACKEND_URL"
            value = "http://${module.saml_auth_proxy_alb.lb_dns_name}:8080/"
          },
          {
            name   = "SAML_PROXY_IDP_METADATA_URL"
            value = var.idp_metadata_url
          },
          {
            name   = "SAML_PROXY_BASE_URL"
            value = var.base_url
          },
        ]
        entryPoint = "/bin/sh",
        command = ["-c",
        # In case the SP_CERT_PATH and SP_KEY_PATH are differnt directories
        <<-EOT
          mkdir -p $(dirname ${SAML_PROXY_SP_CERT_PATH})
          mkdir -p $(dirname ${SAML_PROXY_SP_KEY_PATH})
          echo "${SAML_PROXY_SP_CERT_BYTES}" > "${SP_CERT_PATH}"
          echo "${SAML_PROXY_SP_KEY_BYTES}" > "${SP_KEY_PATH}"
          /usr/bin/saml-auth-proxy
        EOT
        ] 
      }
  ])
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_ecs_service" "saml_auth_proxy" {
  name                               = "saml_auth_proxy"
  launch_type                        = "FARGATE"
  cluster                            = var.ecs_cluster
  task_definition                    = aws_ecs_task_definition.saml_auth_proxy.arn
  desired_count                      = var.loadtest_containers
  deployment_minimum_healthy_percent = 100
  deployment_maximum_percent         = 200

  network_configuration {
    subnets         = var.subnets
    security_groups = var.security_groups
  }
}

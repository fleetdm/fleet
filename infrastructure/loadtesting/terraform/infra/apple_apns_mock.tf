# Mock Apple APNs push server, deployed when var.enable_apple_mdm is true.
#
# Pushing at real Apple infrastructure with thousands of fake device UUIDs
# would get us rate limited, so the only two sane options are "no pushes at
# all" or "pushes to this". local.mdm_apple_environment_variables picks
# whichever matches var.enable_apple_mdm.
#
# This lives in the infra module rather than in its own root module so a single
# terraform apply brings it up alongside the Fleet server it serves. A separate
# root module would mean a second apply, a second state file, and a window
# where Fleet is configured to push at a hostname that does not exist yet.

# ---- ECR + Docker image ----

resource "aws_ecr_repository" "apple_apns_mock" {
  count                = var.enable_apple_mdm ? 1 : 0
  name                 = "${local.customer}-apple-apns-mock"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  force_delete = true
}

resource "docker_image" "apple_apns_mock" {
  count = var.enable_apple_mdm ? 1 : 0
  name  = "${aws_ecr_repository.apple_apns_mock[0].repository_url}:${var.tag}"

  build {
    context    = "${path.module}/../docker/"
    dockerfile = "apple-apns-mock.Dockerfile"
    platform   = "linux/amd64"
    build_args = {
      TAG = var.tag
    }
  }
}

resource "docker_tag" "apple_apns_mock" {
  count        = var.enable_apple_mdm ? 1 : 0
  source_image = docker_image.apple_apns_mock[0].name
  target_image = "${aws_ecr_repository.apple_apns_mock[0].repository_url}:${var.tag}"
}

resource "docker_registry_image" "apple_apns_mock" {
  count         = var.enable_apple_mdm ? 1 : 0
  name          = docker_tag.apple_apns_mock[0].target_image
  keep_remotely = true
}

# ---- CloudWatch Logs ----

resource "aws_cloudwatch_log_group" "apple_apns_mock" {
  count             = var.enable_apple_mdm ? 1 : 0
  name              = "${local.customer}-apple-apns-mock"
  retention_in_days = 30
}

# ---- Security Group ----

resource "aws_security_group" "apple_apns_mock" {
  count       = var.enable_apple_mdm ? 1 : 0
  name_prefix = "${local.customer}-apple-apns-mock-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Apple APNS mock - allows HTTP from internal ALB"

  ingress {
    description     = "HTTP from internal ALB"
    from_port       = local.apple_apns_mock_port
    to_port         = local.apple_apns_mock_port
    protocol        = "tcp"
    security_groups = [aws_security_group.internal.id]
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

# ---- ECS Task Definition ----

resource "aws_ecs_task_definition" "apple_apns_mock" {
  count                    = var.enable_apple_mdm ? 1 : 0
  family                   = "${local.customer}-apple-apns-mock"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 2048
  memory                   = 4096
  execution_role_arn       = module.loadtest.byo-db.byo-ecs.execution_iam_role_arn
  task_role_arn            = module.loadtest.byo-db.byo-ecs.iam_role_arn

  container_definitions = jsonencode([
    {
      name      = "apple-apns-mock"
      image     = docker_registry_image.apple_apns_mock[0].name
      essential = true

      portMappings = [
        {
          containerPort = local.apple_apns_mock_port
          protocol      = "tcp"
        }
      ]

      # One open SSE stream per simulated host, each burning a file
      # descriptor. Fargate's default of 65535 would cap us well short of the
      # 300k hosts this is sized for, so raise it to the platform maximum.
      ulimits = [
        {
          name      = "nofile"
          softLimit = 1048576
          hardLimit = 1048576
        }
      ]

      command = ["--listen", ":${local.apple_apns_mock_port}"]

      # Go does not read the cgroup limit, so without GOMEMLIMIT the GC sizes
      # the heap against the host and lets RSS sail past the task limit until
      # ECS OOM-kills the container -- taking every open SSE stream with it.
      # 90% leaves headroom for stacks and non-heap allocations.
      environment = [
        {
          name  = "GOMEMLIMIT"
          value = "${floor(4096 * 0.9)}MiB"
        }
      ]
      secrets = []

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.apple_apns_mock[0].name
          "awslogs-region"        = data.aws_region.current.region
          "awslogs-stream-prefix" = "apple-apns-mock"
        }
      }
    }
  ])
}

# ---- ECS Service ----

resource "aws_ecs_service" "apple_apns_mock" {
  count = var.enable_apple_mdm ? 1 : 0
  name  = "${local.customer}-apple-apns-mock"
  cluster         = module.loadtest.byo-db.cluster.cluster_name
  task_definition = aws_ecs_task_definition.apple_apns_mock[0].arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    security_groups = [aws_security_group.apple_apns_mock[0].id]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.apple_apns_mock[0].arn
    container_name   = "apple-apns-mock"
    container_port   = local.apple_apns_mock_port
  }

  depends_on = [
    aws_lb_listener_rule.apple_apns_mock,
  ]
}

# ---- ALB target group + host-based listener rule ----

resource "aws_lb_target_group" "apple_apns_mock" {
  count                = var.enable_apple_mdm ? 1 : 0
  name                 = "${local.customer}-apple-apns-mock"
  protocol             = "HTTP"
  port                 = local.apple_apns_mock_port
  target_type          = "ip"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 10

  health_check {
    path                = "/healthz"
    matcher             = "200"
    timeout             = 5
    interval            = 30
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

# The zone is public but the ALB is internal, so this resolves to VPC-private
# addresses; only callers inside the VPC (Fleet, osquery-perf) can reach it.
resource "aws_route53_record" "apple_apns_mock" {
  count   = var.enable_apple_mdm ? 1 : 0
  zone_id = data.aws_route53_zone.main.id
  name    = local.apple_apns_mock_hostname
  type    = "A"

  alias {
    name                   = aws_lb.internal.dns_name
    zone_id                = aws_lb.internal.zone_id
    evaluate_target_health = true
  }
}

# Host-based rather than path-based routing: the mock's paths (/healthz,
# /stats, /events, /3/device/*) would otherwise collide with Fleet's own routes
# on this shared internal ALB -- /healthz in particular is the health check
# path for the Fleet target group in internal_alb.tf. Matching on Host means
# every path on this hostname reaches the mock and nothing else is affected.
resource "aws_lb_listener_rule" "apple_apns_mock" {
  count        = var.enable_apple_mdm ? 1 : 0
  listener_arn = aws_lb_listener.internal.arn
  # Priorities 20/21 on this listener belong to the android_amapi_mock module.
  priority = 30

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.apple_apns_mock[0].arn
  }

  condition {
    host_header {
      values = [local.apple_apns_mock_hostname]
    }
  }
}

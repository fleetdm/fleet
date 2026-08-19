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

# ---- Service discovery (Cloud Map) ----
#
# Simulated devices resolve the mock through this namespace and connect to the
# task IP directly, with no load balancer in the path. See the
# apple_apns_mock_dns_name comment in locals.tf for the ALB connection ceiling
# this exists to escape.
#
# Pure DNS on purpose. ECS Service Connect would also give name resolution, but
# it does it by injecting an Envoy sidecar into every client task and proxying
# through it -- reintroducing exactly the kind of per-connection intermediary
# that broke here, in front of 150k long-lived streams.

resource "aws_service_discovery_private_dns_namespace" "loadtest" {
  count       = var.enable_apple_mdm ? 1 : 0
  name        = local.apple_apns_mock_namespace
  vpc         = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Private DNS for ${local.customer} loadtest services reached without a load balancer"
}

resource "aws_service_discovery_service" "apple_apns_mock" {
  count = var.enable_apple_mdm ? 1 : 0
  name  = "apns-mock"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.loadtest[0].id

    # A records straight to the awsvpc task IP. The TTL is short because the
    # record changes whenever ECS replaces the task, and every reconnecting
    # client re-resolves; Go's resolver does no caching of its own.
    dns_records {
      type = "A"
      ttl  = 15
    }

    # MULTIVALUE returns every healthy task IP in a shuffled order, which is
    # what we would want if this were ever scaled out. It is not -- see the
    # desired_count comment on the ECS service.
    routing_policy = "MULTIVALUE"
  }

  # ECS owns registration and deregistration of task instances. Without this
  # block Cloud Map expects an external health checker and never marks the
  # instance healthy, so the name resolves to nothing.
  health_check_custom_config {
    failure_threshold = 1
  }
}

# ---- Security Group ----

resource "aws_security_group" "apple_apns_mock" {
  count       = var.enable_apple_mdm ? 1 : 0
  name_prefix = "${local.customer}-apple-apns-mock-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Apple APNS mock - allows HTTP from internal ALB"

  # Fleet and osquery-perf share the Fleet service's security groups (the
  # osquery_perf module is handed the same list), so this one rule covers both
  # the push side and the 150k device streams now that they bypass the ALB.
  ingress {
    description     = "HTTP from Fleet and osquery-perf tasks"
    from_port       = local.apple_apns_mock_port
    to_port         = local.apple_apns_mock_port
    protocol        = "tcp"
    security_groups = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].security_groups
  }

  # Retained only so the ALB can reach /healthz for the target group and an
  # operator on the VPN can curl /stats. No device traffic arrives this way.
  ingress {
    description     = "HTTP from internal ALB (operator access and health checks)"
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
  count           = var.enable_apple_mdm ? 1 : 0
  name            = "${local.customer}-apple-apns-mock"
  cluster         = module.loadtest.byo-db.cluster.cluster_name
  task_definition = aws_ecs_task_definition.apple_apns_mock[0].arn
  launch_type     = "FARGATE"

  # Do NOT scale this out. The store is per-process (cmd/apple-apns-mock's
  # store.go holds every pending push and live subscriber in memory), so a push
  # only reaches a device if Fleet's POST /3/device/<token> lands on the same
  # task that is holding that device's SSE stream. With two tasks and both
  # sides picking a task at random from the Cloud Map answer, most pushes would
  # be accepted with a 200 and silently dropped -- a far worse failure than
  # running out of capacity, because nothing reports it. Scaling out needs
  # consistent hashing on the device token in both Fleet's push client and the
  # devices, which does not exist today.
  desired_count = 1

  network_configuration {
    subnets         = data.terraform_remote_state.shared.outputs.vpc.private_subnets
    security_groups = [aws_security_group.apple_apns_mock[0].id]
  }

  # The path devices and Fleet actually use: ECS registers the task IP here as
  # it starts and deregisters it as it stops.
  service_registries {
    registry_arn = aws_service_discovery_service.apple_apns_mock[0].arn
  }

  # Operator access only (/healthz, /stats, /memstats over the VPN). Kept
  # because the Cloud Map name above resolves only inside the VPC, and reaching
  # it from a laptop would need a Route53 Resolver inbound endpoint. This
  # carries no device streams, so it never approaches the per-target connection
  # ceiling that made the ALB the bottleneck in the first place.
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
  name                 = "${local.customer}-apnsm"
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

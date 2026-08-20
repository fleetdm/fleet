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

# ---- Network Load Balancer ----
#
# Its own NLB, not a rule on the shared internal ALB. Every simulated device
# holds an SSE stream open for the whole run (two per macOS host, device plus
# user channel), and an ALB has to hold one connection per stream from its own
# node IPs to a single target IP and port. That tops out near 55k connections
# per ALB node, so a 75k-host run collapsed at roughly 75-100k streams into a
# 39k-count TargetConnectionErrorCount and an equal count of 504s, while the
# mock itself stayed healthy with a target response time near zero.
#
# An NLB fixes that only because preserve_client_ip is set on the target group
# below. Read that comment before changing anything here.

resource "aws_lb" "apple_apns_mock" {
  count              = var.enable_apple_mdm ? 1 : 0
  name               = "${local.customer}-apnsm-nlb"
  internal           = true
  load_balancer_type = "network"
  subnets            = data.terraform_remote_state.shared.outputs.vpc.private_subnets

  # desired_count is 1, so the task lives in exactly one AZ. Without
  # cross-zone, the NLB nodes in the other two AZs have no healthy target and
  # drop out of DNS, funnelling every client onto one node. Enabling it keeps
  # all three nodes usable and the DNS answer stable when ECS replaces the task
  # into a different AZ.
  enable_cross_zone_load_balancing = true

  # An NLB drops idle TCP flows after 350s and that timeout is not
  # configurable, unlike the ALB's. The mock's --keep-alive defaults to 30s and
  # the container command does not override it, so streams stay well inside it.
  # Passing --keep-alive 0 (or anything above 350s) would have every stream
  # silently reset every 350s and 150k clients reconnecting behind it.
}

resource "aws_lb_target_group" "apple_apns_mock" {
  count                = var.enable_apple_mdm ? 1 : 0
  name                 = "${local.customer}-apnsm-nlb"
  protocol             = "TCP"
  port                 = local.apple_apns_mock_port
  target_type          = "ip"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 10

  # THE load-bearing setting. AWS defaults client IP preservation to DISABLED
  # for TCP target groups whose target type is ip, and with it disabled an NLB
  # source-NATs every connection to its own node IPs -- reinstating the exact
  # per-node ephemeral port ceiling (~55k per node against one target IP:port)
  # that made the ALB fall over. With it enabled the NLB does not rewrite the
  # source, so the port space is the osquery-perf side's: each loadtest task IP
  # contributes its own ~28k ephemeral ports toward this target. 150k streams
  # therefore needs at least ~6 osquery-perf containers, which a 75k-host run
  # far exceeds.
  preserve_client_ip = "true"

  # An NLB requires healthy_threshold and unhealthy_threshold to be equal, and
  # caps the HTTP health check timeout at 6s.
  health_check {
    protocol            = "HTTP"
    path                = "/healthz"
    port                = "traffic-port"
    matcher             = "200"
    interval            = 30
    timeout             = 6
    healthy_threshold   = 3
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener" "apple_apns_mock" {
  count             = var.enable_apple_mdm ? 1 : 0
  load_balancer_arn = aws_lb.apple_apns_mock[0].arn
  port              = local.apple_apns_mock_port
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.apple_apns_mock[0].arn
  }
}

# ---- Security Group ----

resource "aws_security_group" "apple_apns_mock" {
  count       = var.enable_apple_mdm ? 1 : 0
  name_prefix = "${local.customer}-apple-apns-mock-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Apple APNS mock - HTTP from loadtest tasks via its own NLB"

  # Because the target group preserves the client IP, packets arrive with the
  # Fleet or osquery-perf task IP as their source, not the NLB's. So the rule
  # has to name the client security groups; a rule naming the load balancer
  # would drop every stream. Fleet and osquery-perf share the Fleet service's
  # security groups (the osquery_perf module is handed the same list), so this
  # one rule covers both the push side and the 150k device streams.
  ingress {
    description     = "HTTP from Fleet and osquery-perf tasks (client IP preserved through the NLB)"
    from_port       = local.apple_apns_mock_port
    to_port         = local.apple_apns_mock_port
    protocol        = "tcp"
    security_groups = module.loadtest.byo-db.byo-ecs.service.network_configuration[0].security_groups
  }

  # Health checks are the one thing that still arrives from the NLB nodes
  # themselves, which draw their addresses from the private subnets. Without
  # this the target group never turns healthy and the listener serves nothing.
  ingress {
    description = "NLB health checks"
    from_port   = local.apple_apns_mock_port
    to_port     = local.apple_apns_mock_port
    protocol    = "tcp"
    cidr_blocks = data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks
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
  # task that is holding that device's SSE stream. A second task would have the
  # NLB spread streams and pushes across both, and most pushes would then be
  # accepted with a 200 and silently dropped -- a far worse failure than
  # running out of capacity, because nothing reports it. Scaling out needs
  # consistent hashing on the device token in both Fleet's push client and the
  # devices, which does not exist today.
  desired_count = 1

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
    aws_lb_listener.apple_apns_mock,
  ]
}

# ---- DNS ----

# The zone is public but the NLB is internal, so this resolves to VPC-private
# addresses; only callers inside the VPC (Fleet, osquery-perf) can reach it.
# Same hostname as before, now pointing at the mock's own NLB instead of a
# host-based rule on the shared internal ALB, which frees listener priority 30.
resource "aws_route53_record" "apple_apns_mock" {
  count   = var.enable_apple_mdm ? 1 : 0
  zone_id = data.aws_route53_zone.main.id
  name    = local.apple_apns_mock_hostname
  type    = "A"

  alias {
    name                   = aws_lb.apple_apns_mock[0].dns_name
    zone_id                = aws_lb.apple_apns_mock[0].zone_id
    evaluate_target_health = true
  }
}

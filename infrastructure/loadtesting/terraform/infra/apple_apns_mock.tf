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

# ---- Redis ----
#
# Dedicated rather than shared with Fleet's: the instances put a full-fleet
# push wave (~900k commands at 300k devices) through this, and running that
# over Fleet's cache would perturb the very system the load test measures.

resource "aws_security_group" "apple_apns_mock_redis" {
  count       = var.enable_apple_mdm ? 1 : 0
  name_prefix = "${local.customer}-apns-mock-redis-"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  description = "Apple APNS mock Redis - allows 6379 from the mock tasks"

  ingress {
    description     = "Redis from the mock APNs tasks"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.apple_apns_mock[0].id]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_elasticache_parameter_group" "apple_apns_mock" {
  count  = var.enable_apple_mdm ? 1 : 0
  name   = "${local.customer}-apns-mock-redis"
  family = "redis7"

  # Every push is fanned out to every instance, so a subscriber that falls
  # behind must not be disconnected for it. Unlimited, as Fleet's cache does.
  parameter {
    name  = "client-output-buffer-limit-pubsub-hard-limit"
    value = "0"
  }
  parameter {
    name  = "client-output-buffer-limit-pubsub-soft-limit"
    value = "0"
  }
  parameter {
    name  = "client-output-buffer-limit-pubsub-soft-seconds"
    value = "0"
  }
}

resource "aws_elasticache_replication_group" "apple_apns_mock" {
  count                       = var.enable_apple_mdm ? 1 : 0
  replication_group_id        = "${local.customer}-apns-mock"
  description                 = "${local.customer} mock APNs coordination"
  engine                      = "redis"
  engine_version              = "7.1"
  node_type                   = var.apple_apns_mock_redis_instance_size
  num_cache_clusters          = var.apple_apns_mock_redis_instance_count
  parameter_group_name        = aws_elasticache_parameter_group.apple_apns_mock[0].id
  subnet_group_name           = data.terraform_remote_state.shared.outputs.vpc.elasticache_subnet_group_name
  security_group_ids          = [aws_security_group.apple_apns_mock_redis[0].id]
  preferred_cache_cluster_azs = slice(["us-east-2a", "us-east-2b", "us-east-2c"], 0, min(var.apple_apns_mock_redis_instance_count, 3))
  port                        = 6379

  # Pending pushes are disposable by design, and failover needs a replica.
  snapshot_retention_limit   = 0
  automatic_failover_enabled = var.apple_apns_mock_redis_instance_count > 1
  at_rest_encryption_enabled = false #tfsec:ignore:aws-elasticache-enable-at-rest-encryption
  transit_encryption_enabled = false #tfsec:ignore:aws-elasticache-enable-in-transit-encryption
  apply_immediately          = true
}

# ---- ECS Task Definition ----

resource "aws_ecs_task_definition" "apple_apns_mock" {
  count                    = var.enable_apple_mdm ? 1 : 0
  family                   = "${local.customer}-apple-apns-mock"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.apple_apns_mock_cpu
  memory                   = var.apple_apns_mock_memory
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

      # Instances coordinate through Redis so Fleet can push to any of them and
      # it reaches whichever holds the device's SSE stream.
      command = [
        "--listen", ":${local.apple_apns_mock_port}",
        "--redis-address", "${aws_elasticache_replication_group.apple_apns_mock[0].primary_endpoint_address}:6379",
        "--redis-key-prefix", "${local.customer}:apns:",
      ]

      # Go does not read the cgroup limit, so without GOMEMLIMIT the GC sizes
      # the heap against the host and lets RSS sail past the task limit until
      # ECS OOM-kills the container -- taking every open SSE stream with it.
      # 90% leaves headroom for stacks and non-heap allocations.
      environment = [
        {
          name  = "GOMEMLIMIT"
          value = "${floor(var.apple_apns_mock_memory * 0.9)}MiB"
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
  desired_count   = var.apple_apns_mock_instance_count
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
  name                 = "${local.customer}-apnsm"
  protocol             = "HTTP"
  port                 = local.apple_apns_mock_port
  target_type          = "ip"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 10

  # SSE streams are one long-lived request each, so "outstanding requests" is
  # the connection count: this spreads devices evenly across instances, where
  # round-robin would only spread new connections.
  load_balancing_algorithm_type = "least_outstanding_requests"

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

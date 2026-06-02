resource "aws_security_group" "internal" {
  name   = "${local.prefix}-int"
  vpc_id = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  ingress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
  egress {
    from_port        = 0
    to_port          = 0
    protocol         = "-1"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }
}

resource "aws_lb" "internal" {
  name     = "${local.prefix}-int"
  internal = true
  security_groups = [
    resource.aws_security_group.internal.id,
  ]
  subnets                    = data.terraform_remote_state.shared.outputs.vpc.private_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
  access_logs {
    bucket  = module.logging_alb.log_s3_bucket_id
    prefix  = local.customer
    enabled = true
  }
}

resource "aws_lb_listener" "internal" {
  load_balancer_arn = resource.aws_lb.internal.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = resource.aws_lb_target_group.internal.arn
  }
}

resource "aws_lb_listener" "internal_https" {
  load_balancer_arn = resource.aws_lb.internal.arn
  port              = 443
  protocol          = "HTTPS"
  certificate_arn   = data.aws_acm_certificate.certificate.arn
  ssl_policy        = "ELBSecurityPolicy-TLS13-1-2-2021-06"

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "Not Found"
      status_code  = "404"
    }
  }
}

resource "aws_lb_target_group" "internal" {
  name                 = "${local.prefix}-int"
  protocol             = "HTTP"
  target_type          = "ip"
  port                 = "80"
  vpc_id               = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  deregistration_delay = 30

  load_balancing_algorithm_type = "least_outstanding_requests"

  health_check {
    path                = "/healthz"
    matcher             = "200"
    timeout             = 10
    interval            = 15
    healthy_threshold   = 5
    unhealthy_threshold = 5
  }
}
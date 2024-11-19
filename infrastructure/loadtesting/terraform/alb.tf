resource "aws_lb" "internal" {
  name                       = "${local.prefix}-internal"
  internal                   = true
  security_groups            = [data.terraform_remote_state.shared.outputs.alb_security_group.id]
  subnets                    = data.terraform_remote_state.shared.outputs.vpc.private_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it
}

resource "aws_lb_listener" "internal" {
  load_balancer_arn = aws_lb.internal.arn
  port              = 80
  protocol          = "HTTP" #tfsec:ignore:aws-elb-http-not-used

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.internal.arn
  }
}

resource "aws_lb_listener_rule" "main" {
  listener_arn = data.terraform_remote_state.shared.outputs.alb-listener.arn

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  condition {
    host_header {
      values = ["${terraform.workspace}.loadtest.fleetdm.com"]
    }
  }
}

resource "aws_lb_target_group" "internal" {
  name                 = "${local.prefix}-internal"
  protocol             = "HTTP"
  target_type          = "ip"
  port                 = "8080"
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

resource "aws_lb_target_group" "main" {
  name                 = local.prefix
  protocol             = "HTTP"
  target_type          = "ip"
  port                 = "8080"
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


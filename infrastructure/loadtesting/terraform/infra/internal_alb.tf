resource "aws_lb" "internal" {
  name                       = "${local.prefix}-internal"
  internal                   = true
  security_groups            = [
    data.terraform_remote_state.shared.outputs.alb_security_group.id, 
    try(module.loadtest.byo-db.alb.security_group_id, ""),
  ]
  subnets                    = data.terraform_remote_state.shared.outputs.vpc.private_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it
}

resource "aws_lb_listener" "internal" {
  load_balancer_arn = resource.aws_lb.internal.arn
  port              = 80
  protocol          = "HTTP" #tfsec:ignore:aws-elb-http-not-used

  default_action {
    type             = "forward"
    target_group_arn = resource.aws_lb_target_group.internal.arn
  }
}

resource "aws_lb_target_group" "internal" {
  name                 = "${local.prefix}-internal"
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


# what if migrations had an output that said migration_executed = true???, If true, then we enable the target group?
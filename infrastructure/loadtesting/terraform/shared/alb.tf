resource "aws_alb" "main" {
  name                       = "fleetdm"
  internal                   = false #tfsec:ignore:aws-elb-alb-not-public
  security_groups            = [aws_security_group.lb.id]
  subnets                    = module.vpc.public_subnets
  idle_timeout               = 905
  drop_invalid_header_fields = true
  #checkov:skip=CKV_AWS_150:don't like it
}

resource "aws_alb_listener" "https-fleetdm" {
  load_balancer_arn = aws_alb.main.arn
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate_validation.wildcard.certificate_arn

  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "moved to subdomains, try https://default.loadtest.fleetdm.com"
      status_code  = "404"
    }
  }
}

resource "aws_alb_listener" "http" {
  load_balancer_arn = aws_alb.main.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

# Security group for the public internet facing load balancer
resource "aws_security_group" "lb" {
  name        = "${local.prefix} load balancer"
  description = "${local.prefix} Load balancer security group"
  vpc_id      = module.vpc.vpc_id
}

# Allow traffic from public internet
resource "aws_security_group_rule" "lb-ingress" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "443"
  to_port     = "443"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-ingress-sgr

  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-http-ingress" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "80"
  to_port     = "80"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-ingress-sgr

  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-es" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "9200"
  to_port     = "9200"
  protocol    = "tcp"
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = aws_security_group.lb.id
}
resource "aws_security_group_rule" "lb-es-apm" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "8200"
  to_port     = "8200"
  protocol    = "tcp"
  cidr_blocks = concat(["10.0.0.0/8"], [for ip in module.vpc.nat_public_ips : "${ip}/32"])

  security_group_id = aws_security_group.lb.id
}
resource "aws_security_group_rule" "lb-kibana" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "5601"
  to_port     = "5601"
  protocol    = "tcp"
  cidr_blocks = ["10.0.0.0/8"]

  security_group_id = aws_security_group.lb.id
}

# Allow outbound traffic
resource "aws_security_group_rule" "lb-egress" {
  description = "${local.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr

  security_group_id = aws_security_group.lb.id
}


# Security group for the public internet facing load balancer
resource "aws_security_group" "lb" {
  name        = "${var.prefix} load balancer"
  description = "${var.prefix} Load balancer security group"
  vpc_id      = module.vpc.vpc_id
}

# Allow traffic from public internet
resource "aws_security_group_rule" "lb-ingress" {
  description = "${var.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "443"
  to_port     = "443"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-http-ingress" {
  description = "${var.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port   = "80"
  to_port     = "80"
  protocol    = "tcp"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.lb.id
}

# Allow outbound traffic
resource "aws_security_group_rule" "lb-egress" {
  description = "${var.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.lb.id
}

# Security group for the backends that run the application.
# Allows traffic from the load balancer
resource "aws_security_group" "backend" {
  name        = "${var.prefix} backend"
  description = "${var.prefix} Backend security group"
  vpc_id      = module.vpc.vpc_id

}

# Allow traffic from the load balancer to the backends
resource "aws_security_group_rule" "backend-ingress" {
  description = "${var.prefix}: allow traffic from load balancer"
  type        = "ingress"

  from_port                = "8080"
  to_port                  = "8080"
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.lb.id
  security_group_id        = aws_security_group.backend.id
}

# Allow outbound traffic from the backends
resource "aws_security_group_rule" "backend-egress" {
  description = "${var.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"]

  security_group_id = aws_security_group.backend.id
}
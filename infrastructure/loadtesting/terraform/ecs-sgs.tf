# Security group for the backends that run the application.
# Allows traffic from the load balancer
resource "aws_security_group" "backend" {
  name        = "${local.prefix} backend"
  description = "${local.prefix} Backend security group"
  vpc_id      = data.terraform_remote_state.shared.outputs.vpc.vpc_id

}

# Allow traffic from the load balancer to the backends
resource "aws_security_group_rule" "backend-ingress" {
  description = "${local.prefix}: allow traffic from load balancer"
  type        = "ingress"

  from_port                = "8080"
  to_port                  = "8080"
  protocol                 = "tcp"
  source_security_group_id = data.terraform_remote_state.shared.outputs.alb_security_group.id
  security_group_id        = aws_security_group.backend.id
}

# Allow outbound traffic from the backends
resource "aws_security_group_rule" "backend-egress" {
  description = "${local.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr

  security_group_id = aws_security_group.backend.id
}

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

  from_port = "443"
  to_port   = "443"
  protocol  = "tcp"
  // Internet connectivity here is by design
  cidr_blocks       = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-ingress-sgr
  security_group_id = aws_security_group.lb.id
}

resource "aws_security_group_rule" "lb-http-ingress" {
  description = "${local.prefix}: allow traffic from public internet"
  type        = "ingress"

  from_port = "80"
  to_port   = "80"
  protocol  = "tcp"
  // Internet connectivity here is by design
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
// Egress filtering is not currently provided by our Terraform templates.
resource "aws_security_group_rule" "lb-egress" { #tfsec:ignore:aws-vpc-no-public-egress-sgr:exp:2022-10-01
  description = "${local.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port   = 0
  to_port     = 0
  protocol    = "-1"
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr

  security_group_id = aws_security_group.lb.id
}

# Security group for the backends that run the application.
# Allows traffic from the load balancer
resource "aws_security_group" "backend" {
  name        = "${local.prefix} backend"
  description = "${local.prefix} Backend security group"
  vpc_id      = module.vpc.vpc_id

}

# Allow traffic from the load balancer to the backends
resource "aws_security_group_rule" "backend-ingress" {
  description = "${local.prefix}: allow traffic from load balancer"
  type        = "ingress"

  from_port                = "8080"
  to_port                  = "8080"
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.lb.id
  security_group_id        = aws_security_group.backend.id
}

# Allow outbound traffic from the backends
resource "aws_security_group_rule" "backend-egress" {
  description = "${local.prefix}: allow all outbound traffic"
  type        = "egress"

  from_port = 0
  to_port   = 0
  protocol  = "-1"
  // Egress filtering is not currently provided by our Terraform templates.
  cidr_blocks = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr:exp:2022-10-01

  security_group_id = aws_security_group.backend.id
}

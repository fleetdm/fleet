resource "aws_elasticache_replication_group" "default" {
  availability_zones            = ["us-east-2a", "us-east-2b", "us-east-2c"]
  engine                        = "redis"
  parameter_group_name          = aws_elasticache_parameter_group.default.id
  subnet_group_name             = module.vpc.elasticache_subnet_group_name
  security_group_ids            = [aws_security_group.redis.id, aws_security_group.backend.id]
  replication_group_id          = "fleetdm-redis"
  number_cache_clusters         = 3
  node_type                     = "cache.m6g.large"
  engine_version                = "5.0.6"
  port                          = "6379"
  snapshot_retention_limit      = 0
  automatic_failover_enabled    = true
  at_rest_encryption_enabled    = true
  transit_encryption_enabled    = true
  apply_immediately             = true
  replication_group_description = "fleetdm-redis"

}

resource "aws_elasticache_parameter_group" "default" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  name   = "fleetdm-redis-foobar"
  family = "redis5.0"

  parameter {
    name  = "client-output-buffer-limit-pubsub-hard-limit"
    value = "0"
  }
  parameter {
    name  = "client-output-buffer-limit-pubsub-soft-limit"
    value = "0"
  }
}

resource "aws_security_group" "redis" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key tfsec:ignore:aws-vpc-add-description-to-security-group
  name   = local.security_group_name
  description = "Security group for Redis"
  vpc_id = module.vpc.vpc_id
}

locals {
  security_group_name = "${local.prefix}-elasticache-redis"
}

resource "aws_security_group_rule" "ingress" {
  description       = "Redis from private VPC"
resource "aws_security_group_rule" "ingress" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  type              = "ingress"
  from_port         = "6379"
  to_port           = "6379"
  protocol          = "tcp"
  cidr_blocks       = module.vpc.private_subnets_cidr_blocks
  security_group_id = aws_security_group.redis.id
}

resource "aws_security_group_rule" "egress" {
  description       = "Redis VPC egress"
resource "aws_security_group_rule" "egress" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  // Egress filtering is not currently provided by our Terraform templates.
  cidr_blocks       = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr:exp:2022-10-01
  security_group_id = aws_security_group.redis.id
}

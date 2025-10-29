resource "aws_elasticache_replication_group" "default" {
  preferred_cache_cluster_azs = ["us-east-2a", "us-east-2b", "us-east-2c"]
  engine                      = "redis"
  parameter_group_name        = aws_elasticache_parameter_group.default.id
  subnet_group_name           = data.terraform_remote_state.shared.outputs.vpc.elasticache_subnet_group_name
  security_group_ids          = [aws_security_group.redis.id, aws_security_group.backend.id]
  replication_group_id        = "${local.prefix}-redis"
  num_cache_clusters          = 3
  node_type                   = var.redis_instance_type
  engine_version              = "6.2"
  port                        = "6379"
  snapshot_retention_limit    = 0
  automatic_failover_enabled  = true
  at_rest_encryption_enabled  = false #tfsec:ignore:aws-elasticache-enable-at-rest-encryption
  transit_encryption_enabled  = false #tfsec:ignore:aws-elasticache-enable-in-transit-encryption
  apply_immediately           = true
  description                 = "${local.prefix}-redis"

}

resource "aws_elasticache_parameter_group" "default" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  name   = "${local.prefix}-redis"
  family = "redis6.x"

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
  vpc_id = data.terraform_remote_state.shared.outputs.vpc.vpc_id
}

locals {
  security_group_name = "${local.prefix}-elasticache-redis"
}

resource "aws_security_group_rule" "ingress" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  type              = "ingress"
  from_port         = "6379"
  to_port           = "6379"
  protocol          = "tcp"
  cidr_blocks       = concat(data.terraform_remote_state.shared.outputs.vpc.private_subnets_cidr_blocks, local.vpn_cidr_blocks)
  security_group_id = aws_security_group.redis.id
}

resource "aws_security_group_rule" "egress" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"] #tfsec:ignore:aws-vpc-no-public-egress-sgr
  security_group_id = aws_security_group.redis.id
}

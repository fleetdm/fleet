resource "aws_elasticache_replication_group" "main" {
  availability_zones         = ["us-east-2a", "us-east-2b", "us-east-2c"]
  engine                     = "redis"
  parameter_group_name       = aws_elasticache_parameter_group.main.id
  subnet_group_name          = var.vpc.elasticache_subnet_group_name
  security_group_ids         = [aws_security_group.redis.id]
  replication_group_id       = var.prefix
  num_cache_clusters         = 3
  node_type                  = "cache.m6g.large"
  engine_version             = "5.0.6"
  port                       = "6379"
  snapshot_retention_limit   = 0
  automatic_failover_enabled = true
  at_rest_encryption_enabled = false #tfsec:ignore:aws-elasticache-enable-at-rest-encryption
  transit_encryption_enabled = false #tfsec:ignore:aws-elasticache-enable-in-transit-encryption
  apply_immediately          = true
  description                = var.prefix

}

resource "aws_elasticache_parameter_group" "main" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  name   = var.prefix
  family = "redis5.0"

  parameter {
    name  = "client-output-buffer-limit-pubsub-hard-limit"
    value = "0"
  }
  parameter {
    name  = "client-output-buffer-limit-pubsub-soft-limit"
    value = "0"
  }

  parameter {
    name  = "databases"
    value = "65536"
  }
}

resource "aws_security_group" "redis" { #tfsec:ignore:aws-cloudwatch-log-group-customer-key tfsec:ignore:aws-vpc-add-description-to-security-group
  name   = var.prefix
  vpc_id = var.vpc.vpc_id
}

resource "aws_security_group_rule" "ingress" { #tfsec:ignore:aws-vpc-add-description-to-security-group-rule
  type              = "ingress"
  from_port         = "6379"
  to_port           = "6379"
  protocol          = "tcp"
  cidr_blocks       = var.vpc.private_subnets_cidr_blocks
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

output "redis_cluster" {
  value = aws_elasticache_replication_group.main
}

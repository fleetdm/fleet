resource "aws_elasticache_replication_group" "main" {
  preferred_cache_cluster_azs = ["us-east-2a", "us-east-2b", "us-east-2c"]
  engine                      = "redis"
  parameter_group_name        = aws_elasticache_parameter_group.main.id
  subnet_group_name           = var.vpc.elasticache_subnet_group_name
  security_group_ids          = [aws_security_group.redis.id]
  replication_group_id        = var.prefix
  num_cache_clusters          = 3
  node_type                   = "cache.m6g.large"
  engine_version              = "5.0.6"
  port                        = "6379"
  snapshot_retention_limit    = 0
  automatic_failover_enabled  = true
  at_rest_encryption_enabled  = false #tfsec:ignore:aws-elasticache-enable-at-rest-encryption
  transit_encryption_enabled  = false #tfsec:ignore:aws-elasticache-enable-in-transit-encryption
  apply_immediately           = true
  description                 = var.prefix

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
  name        = "${var.prefix}-redis"
  vpc_id      = var.vpc.vpc_id
  description = "${var.prefix}-redis"

  ingress {
    from_port   = 6379
    to_port     = 6397
    protocol    = "TCP"
    cidr_blocks = var.vpc.private_subnets_cidr_blocks
  }
}

output "redis_cluster" {
  value = aws_elasticache_replication_group.main
}

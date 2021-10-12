variable "maintenance_window" {
  default = ""
}
variable "engine_version" {
  default = "6.x"
}
variable "number_cache_clusters" {
  default = 3
}
variable "redis_instance" {
  default = "cache.m5.large"
}
resource "aws_elasticache_replication_group" "default" {
  availability_zones            = ["us-east-2a", "us-east-2b", "us-east-2c"]
  engine                        = "redis"
  parameter_group_name          = "default.redis6.x"
  subnet_group_name             = module.vpc.elasticache_subnet_group_name
  security_group_ids            = [aws_security_group.redis.id]
  replication_group_id          = "fleetdm-redis"
  number_cache_clusters         = var.number_cache_clusters
  node_type                     = var.redis_instance
  engine_version                = var.engine_version
  port                          = "6379"
  maintenance_window            = var.maintenance_window
  snapshot_retention_limit      = 0
  automatic_failover_enabled    = false
  at_rest_encryption_enabled    = false
  transit_encryption_enabled    = false
  apply_immediately             = true
  replication_group_description = "fleetdm-redis"
}

resource "aws_security_group" "redis" {
  name   = local.security_group_name
  vpc_id = module.vpc.vpc_id
}

locals {
  security_group_name = "${var.prefix}-elasticache-redis"
}

resource "aws_security_group_rule" "ingress" {
  type              = "ingress"
  from_port         = "6379"
  to_port           = "6379"
  protocol          = "tcp"
  cidr_blocks       = module.vpc.private_subnets_cidr_blocks
  security_group_id = aws_security_group.redis.id
}

resource "aws_security_group_rule" "egress" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.redis.id
}


resource "aws_efs_file_system" "vuln" {}

resource "aws_security_group" "efs_security_group" {
  name_prefix = "${var.customer_prefix}-efs-mount-sg"
  vpc_id      = var.vpc_id

  // NFS
  ingress {
    from_port       = 2049
    to_port         = 2049
    protocol        = "tcp"
    security_groups = var.fleet_config.networking.security_groups # Allow traffic from the ECS task security group
  }
}

resource "aws_efs_mount_target" "vuln" {
  for_each       = var.fleet_config.networking.subnets
  file_system_id = aws_efs_file_system.vuln.id
  subnet_id      = each.value
}
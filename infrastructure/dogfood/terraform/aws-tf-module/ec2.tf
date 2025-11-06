resource "aws_instance" "fleetbot" {
  ami                    = "ami-0d1b5a8c13042c939" # us-east-2 - Ubuntu 24.04 LTS
  instance_type          = "t3.medium"
  subnet_id              = module.main.vpc.private_subnets[0] # take the first subnet in the list
  vpc_security_group_ids = [aws_security_group.fleetbot.id]

  root_block_device {
    volume_size = 50
  }

  key_name = "dogfood-fleetbot"

  tags = {
    Name = "${local.customer}-fleetbot"
  }

  volume_tags = merge({ Name = "${local.customer}-fleetbot" })
}

resource "aws_security_group" "fleetbot" {
  name        = "${local.customer}-fleetbot-sg"
  description = "Primary security group for ${local.customer}-fleetbot (SSH)"
  vpc_id      = module.main.vpc.vpc_id

  tags = {
    Name = "${local.customer}-fleetbot-sg"
  }
}

resource "aws_vpc_security_group_ingress_rule" "fleetbot_ssh" {
  security_group_id = aws_security_group.fleetbot.id
  cidr_ipv4         = "10.255.0.0/16"
  from_port         = 22
  to_port           = 22
  ip_protocol       = "tcp"
}

resource "aws_vpc_security_group_egress_rule" "fleetbot_any" {
  security_group_id = aws_security_group.fleetbot.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

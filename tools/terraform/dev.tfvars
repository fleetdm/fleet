scaling_configuration = {
  auto_pause               = true
  min_capacity             = 2
  max_capacity             = 16
  seconds_until_auto_pause = 300
  timeout_action           = "ForceApplyCapacityChange"
}

engine_mode   = "serverless"
http_endpoint = true

subnets = ["subnet-7ee43033", "subnet-db4f37b2", "subnet-12f67069"]

vpc_id = "vpc-1a533973"

allowed_security_groups = "sg-8c6667e5"

vpc_security_group_ids = ["sg-8c6667e5"]

allowed_cidr_blocks = ["172.31.16.0/20"]

cert_arn = "arn:aws:acm:us-east-2:123169442427:certificate/119ae69d-b1da-4e88-b849-e9b75ee173a0"
module "vpc" {
  source = "terraform-aws-modules/vpc/aws"

  name = "fleet-vpc"
  cidr = "10.10.0.0/16"

  azs                 = ["us-east-2a", "us-east-2b", "us-east-2c"]
  private_subnets     = ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"]
  public_subnets      = ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"]
  database_subnets    = ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"]
  elasticache_subnets = ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"]

  create_database_subnet_group       = true
  create_database_subnet_route_table = true

  create_elasticache_subnet_group       = true
  create_elasticache_subnet_route_table = true

  enable_vpn_gateway     = false
  one_nat_gateway_per_az = false

  single_nat_gateway = true
  enable_nat_gateway = true
}
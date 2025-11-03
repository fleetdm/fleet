module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "fleet-vpc"
  cidr = "10.12.0.0/16"

  azs                 = ["us-east-2a", "us-east-2b", "us-east-2c"]
  private_subnets     = ["10.12.1.0/24", "10.12.2.0/24", "10.12.3.0/24"]
  public_subnets      = ["10.12.11.0/24", "10.12.12.0/24", "10.12.13.0/24"]
  database_subnets    = ["10.12.21.0/24", "10.12.22.0/24", "10.12.23.0/24"]
  elasticache_subnets = ["10.12.31.0/24", "10.12.32.0/24", "10.12.33.0/24"]

  create_database_subnet_group       = true
  create_database_subnet_route_table = true

  create_elasticache_subnet_group       = true
  create_elasticache_subnet_route_table = true

  enable_vpn_gateway     = false
  one_nat_gateway_per_az = false

  single_nat_gateway   = true
  enable_nat_gateway   = true
  enable_dns_hostnames = true

  # Tags required for EKS - role tags are required on subnets
  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }

  # Note: Kubernetes cluster-specific tags are added by the signoz module
  # when creating each EKS cluster, not at the VPC level
  tags = {
    "shared" = "true"
  }
}

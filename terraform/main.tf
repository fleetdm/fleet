module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.18.1"

  name = var.vpc.name
  cidr = var.vpc.cidr

  azs                                   = var.vpc.azs
  private_subnets                       = var.vpc.private_subnets
  public_subnets                        = var.vpc.public_subnets
  database_subnets                      = var.vpc.database_subnets
  elasticache_subnets                   = var.vpc.elasticache_subnets
  create_database_subnet_group          = var.vpc.create_database_subnet_group
  create_database_subnet_route_table    = var.vpc.create_database_subnet_route_table
  create_elasticache_subnet_group       = var.vpc.create_elasticache_subnet_group
  create_elasticache_subnet_route_table = var.vpc.create_elasticache_subnet_route_table
  enable_vpn_gateway                    = var.vpc.enable_vpn_gateway
  one_nat_gateway_per_az                = var.vpc.one_nat_gateway_per_az
  single_nat_gateway                    = var.vpc.single_nat_gateway
  enable_nat_gateway                    = var.vpc.enable_nat_gateway
}

module "byo-vpc" {
  source = "./byo-vpc"
  vpc_config = {
    vpc_id = module.vpc.vpc_id
    networking = {
      subnets = module.vpc.private_subnets
    }
  }
  rds_config = merge(var.rds_config, {
    subnets = module.vpc.database_subnets
  })
  redis_config = merge(var.redis_config, {
    subnets                       = module.vpc.elasticache_subnets
    elasticache_subnet_group_name = module.vpc.elasticache_subnet_group_name
    availability_zones            = var.vpc.azs
  })
  alb_config = merge(var.alb_config, {
    subnets         = module.vpc.public_subnets
    certificate_arn = var.certificate_arn
  })
  ecs_cluster  = var.ecs_cluster
  fleet_config = var.fleet_config
}

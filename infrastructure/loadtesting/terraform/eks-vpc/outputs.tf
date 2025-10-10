output "vpc" {
  description = "VPC module outputs for EKS"
  value = {
    vpc_id             = module.vpc.vpc_id
    private_subnets    = module.vpc.private_subnets
    public_subnets     = module.vpc.public_subnets
    vpc_cidr_block     = module.vpc.vpc_cidr_block
    nat_gateway_ids    = module.vpc.natgw_ids
    azs                = module.vpc.azs
  }
}

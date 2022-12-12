module "ecs" {
  source           = "./byo-ecs"
  ecs_cluster      = module.cluster.cluster_name
  fleet_config     = var.fleet_config
  migration_config = var.migration_config
  vpc_id           = var.vpc_id
}

module "cluster" {
  source  = "terraform-aws-modules/ecs/aws"
  version = "4.1.2"

  autoscaling_capacity_providers        = var.ecs_cluster.autoscaling_capacity_providers
  cluster_configuration                 = var.ecs_cluster.cluster_configuration
  cluster_name                          = var.ecs_cluster.cluster_name
  cluster_settings                      = var.ecs_cluster.cluster_settings
  create                                = var.ecs_cluster.create
  default_capacity_provider_use_fargate = var.ecs_cluster.default_capacity_provider_use_fargate
  fargate_capacity_providers            = var.ecs_cluster.fargate_capacity_providers
  tags                                  = var.ecs_cluster.tags
}

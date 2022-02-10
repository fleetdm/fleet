

module "safer-mysql-db" {
  source               = "GoogleCloudPlatform/sql-db/google//modules/safer_mysql"
  version              = "9.0.0"
  name                 = "${var.prefix}-mysql"
  random_instance_name = true
  project_id           = var.project_id

  deletion_protection = false

  database_version = var.db_version
  region           = var.region
  zone             = var.db_zone
  tier             = var.db_tier
  user_name        = var.db_user

  vpc_network = module.vpc.network_self_link

  // Optional: used to enforce ordering in the creation of resources.
  module_depends_on = [module.private-service-access.peering_completed]
}
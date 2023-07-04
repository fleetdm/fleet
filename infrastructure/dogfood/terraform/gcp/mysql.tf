resource "random_password" "fleet-db-user-pw" {
  length = 12
}

module "fleet-mysql" {
  source               = "GoogleCloudPlatform/sql-db/google//modules/mysql"
  version              = "9.0.0"
  name                 = "${var.prefix}-mysql"
  random_instance_name = true
  project_id           = var.project_id

  deletion_protection = false

  additional_users = [
    {
      name     = var.db_user
      password = random_password.fleet-db-user-pw.result
      host     = "% (any host)"
      type     = "BUILT_IN"
    }
  ]

  ip_configuration = {
    ipv4_enabled = false
    # We never set authorized networks, we need all connections via the
    # public IP to be mediated by Cloud SQL.
    authorized_networks = []
    require_ssl         = false
    private_network     = module.vpc.network_self_link
  }

  database_version = var.db_version
  region           = var.region
  zone             = var.db_zone
  tier             = var.db_tier
  additional_databases = [
    {
      name      = var.db_name
      charset   = "utf8mb4"
      collation = "utf8mb4_unicode_ci"
    }
  ]


  // Optional: used to enforce ordering in the creation of resources.
  module_depends_on = [module.private-service-access.peering_completed]
}

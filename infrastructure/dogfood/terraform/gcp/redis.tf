resource "google_redis_instance" "cache" {
  name               = "${var.prefix}-redis"
  tier               = "STANDARD_HA"
  memory_size_gb     = var.redis_mem
  authorized_network = module.vpc.network_name
  connect_mode       = "PRIVATE_SERVICE_ACCESS"
  display_name       = "${var.prefix}-redis"
  depends_on         = [module.private-service-access.peering_completed]
}
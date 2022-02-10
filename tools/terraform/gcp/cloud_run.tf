resource "google_compute_region_network_endpoint_group" "neg" {
  name                  = "${var.prefix}-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_service.default.name
  }
}

resource "google_cloud_run_service" "default" {
  name     = "${var.prefix}-backend"
  location = var.region

  template {
    spec {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/hello"
        ports {
          name           = "http1"
          container_port = 8080
        }
        env {
          name  = "FLEET_MYSQL_USERNAME"
          value = var.db_user
        }
        env {
          name  = "FLEET_MYSQL_PASSWORD"
          value = module.safer-mysql-db.generated_user_password
        }
        env {
          name  = "FLEET_SERVER_TLS"
          value = false
        }
        env {
          name  = "FLEET_MYSQL_ADDRESS"
          value = module.safer-mysql-db.instance_connection_name
        }
        env {
          name  = "FLEET_REDIS_ADDRESS"
          value = google_redis_instance.cache.host
        }
        env {
          name  = "FLEET_REDIS_PORT"
          value = google_redis_instance.cache.port
        }

      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale"         = "1000"
        "run.googleapis.com/cloudsql-instances"    = module.safer-mysql-db.instance_connection_name
        "run.googleapis.com/vpc-access-connector"  = tolist(module.serverless-connector.connector_ids)[0]
#        "run.googleapis.com/execution-environment" = "gen2"
        "run.googleapis.com/ingress"               = "internal-and-cloud-load-balancing"
        "run.googleapis.com/ingress-status"        = "internal-and-cloud-load-balancing"
        "run.googleapis.com/vpc-access-egress"     = "all"
        "run.googleapis.com/client-name"           = "terraform"
      }
    }
  }
  autogenerate_revision_name = true
}

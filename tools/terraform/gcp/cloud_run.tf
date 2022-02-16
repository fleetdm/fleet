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
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.my-repo.name}/${var.image}"
        ports {
          name           = "http1"
          container_port = 8080
        }
        env {
          name  = "FLEET_MYSQL_USERNAME"
          value = var.db_user
        }
        env {
          name  = "FLEET_MYSQL_DATABASE"
          value = var.db_name
        }
        env {
          name  = "FLEET_MYSQL_PASSWORD"
          value = random_password.fleet-db-user-pw.result
        }
        env {
          name  = "FLEET_SERVER_TLS"
          value = false
        }
        env {
          name  = "FLEET_MYSQL_ADDRESS"
          value = module.fleet-mysql.private_ip_address
        }
        env {
          name  = "FLEET_REDIS_ADDRESS"
          value = google_redis_instance.cache.host
        }
        env {
          name  = "FLEET_REDIS_PORT"
          value = google_redis_instance.cache.port
        }
        command = ["fleet", "prepare", "--no-prompt=true", "db"]
#        command = ["/bin/sh", "-c"]
#        args = [
#          "fleet prepare --no-prompt=true db && fleet serve"
#        ]
      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale"        = "1000"
        "run.googleapis.com/cloudsql-instances"   = module.fleet-mysql.instance_connection_name
        "run.googleapis.com/vpc-access-connector" = tolist(module.serverless-connector.connector_ids)[0]
        #        "run.googleapis.com/execution-environment" = "gen2"
        "run.googleapis.com/ingress"           = "internal-and-cloud-load-balancing"
        "run.googleapis.com/ingress-status"    = "internal-and-cloud-load-balancing"
        "run.googleapis.com/vpc-access-egress" = "all"
        "run.googleapis.com/client-name"       = "terraform"
      }
    }
  }
  autogenerate_revision_name = true
}

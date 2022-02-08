resource "google_cloud_run_service" "default" {
  name     = "fleetdm"
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
          value = "root"
        }
        env {
          name  = "FLEET_MYSQL_PASSWORD"
          value = "root"
        }
        env {
          name  = "FLEET_SERVER_TLS"
          value = "root"
        }
        env {
          name  = "FLEET_MYSQL_ADDRESS"
          value = "root"
        }
        env {
          name  = "FLEET_REDIS_ADDRESS"
          value = "root"
        }

      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale"        = "1000"
        "run.googleapis.com/cloudsql-instances"   = google_sql_database_instance.instance.connection_name
        "run.googleapis.com/vpc-access-connector" = module.serverless-connector.connector_ids
        "run.googleapis.com/ingress"              = "internal-and-cloud-load-balancing"
        "run.googleapis.com/ingress-status"       = "internal-and-cloud-load-balancing"
        "run.googleapis.com/vpc-access-egress"    = "all"
        "run.googleapis.com/client-name"          = "terraform"
      }
    }
  }
  autogenerate_revision_name = true
}

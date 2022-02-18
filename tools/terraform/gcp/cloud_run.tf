resource "google_compute_region_network_endpoint_group" "neg" {
  name                  = "${var.prefix}-neg"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  cloud_run {
    service = google_cloud_run_service.default.name
  }
}

data "google_iam_policy" "noauth" {
  binding {
    role = "roles/run.invoker"
    members = [
      "allUsers",
    ]
  }
}

resource "google_cloud_run_service_iam_policy" "noauth" {
  location = google_cloud_run_service.default.location
  project  = google_cloud_run_service.default.project
  service  = google_cloud_run_service.default.name

  policy_data = data.google_iam_policy.noauth.policy_data
}

resource "random_pet" "suffix" {
  length = 1
}

resource "google_secret_manager_secret" "secret" {
  secret_id = "fleet-db-password-${random_pet.suffix.id}"
  replication {
    automatic = true
  }
}

resource "google_secret_manager_secret_version" "secret-version-data" {
  secret      = google_secret_manager_secret.secret.name
  secret_data = module.fleet-mysql.generated_user_password
}

data "google_compute_default_service_account" "default" {}

resource "google_secret_manager_secret_iam_member" "secret-access" {
  secret_id  = google_secret_manager_secret.secret.id
  role       = "roles/secretmanager.secretAccessor"
  member     = "serviceAccount:${data.google_compute_default_service_account.default.email}"
  depends_on = [google_secret_manager_secret.secret]
}

resource "google_cloud_run_service" "default" {
  name     = "${var.prefix}-backend"
  location = var.region
  metadata {
    annotations = {
      "run.googleapis.com/ingress"        = "internal-and-cloud-load-balancing"
      "run.googleapis.com/ingress-status" = "internal-and-cloud-load-balancing"
    }
  }


  template {
    spec {
      containers {
        resources {
          limits = {
            cpu    = var.fleet_cpu
            memory = var.fleet_memory
          }
        }
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
          name  = "FLEET_SERVER_TLS"
          value = false
        }
        env {
          name  = "FLEET_MYSQL_ADDRESS"
          value = module.fleet-mysql.private_ip_address
        }
        env {
          name  = "FLEET_REDIS_ADDRESS"
          value = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
        }
        env {
          name = "FLEET_MYSQL_PASSWORD"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.secret.secret_id
              key  = "latest"
            }
          }
        }
        command = ["/bin/sh"]
        args = [
          "-c",
          "fleet prepare --no-prompt=true db; exec fleet serve"
        ]
      }
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/minScale"        = "1"
        "autoscaling.knative.dev/maxScale"        = "1000"
        "run.googleapis.com/cloudsql-instances"   = module.fleet-mysql.instance_connection_name
        "run.googleapis.com/vpc-access-connector" = tolist(module.serverless-connector.connector_ids)[0]
        "run.googleapis.com/vpc-access-egress"    = "all-traffic"
        "run.googleapis.com/client-name"          = "terraform"
        "run.googleapis.com/cpu-throttling"       = "false"

      }
    }
  }
  autogenerate_revision_name = true
}

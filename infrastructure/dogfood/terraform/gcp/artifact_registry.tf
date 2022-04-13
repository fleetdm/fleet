resource "google_artifact_registry_repository" "my-repo" {
  provider      = google-beta
  location      = var.region
  repository_id = "${var.prefix}-repository"
  description   = "repository to hold fleet container images for cloud run"
  format        = "DOCKER"
}
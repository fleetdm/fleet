resource "google_project_service" "vpcaccess-api" {
  project = var.project_id # Replace this with your project ID in quotes
  service = "vpcaccess.googleapis.com"
}

resource "google_project_service" "secretmanager" {
  provider = google-beta
  project  = var.project_id
  service  = "secretmanager.googleapis.com"
}
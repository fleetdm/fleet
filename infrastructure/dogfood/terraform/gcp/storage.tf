data "google_client_config" "current" {}

resource "google_service_account" "service_account" {
  account_id = "fleet-svc"
}

resource "google_storage_hmac_key" "key" {
  service_account_email = google_service_account.service_account.email
}

resource "google_storage_bucket" "software_installers" {
  name          = var.software_installers_bucket_name
  location      = data.google_client_config.current.region
  force_destroy = true

  uniform_bucket_level_access = true
}

resource "google_storage_bucket_iam_member" "hmac_sa_storage_admin" {
  bucket = google_storage_bucket.software_installers.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.service_account.email}"
}
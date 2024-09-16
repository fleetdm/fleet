terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.51.0"
    }
    google-beta = {
      source = "hashicorp/google-beta"
    }
  }
}

provider "google" {
  # Configuration options
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}
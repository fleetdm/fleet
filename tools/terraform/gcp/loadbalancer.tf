module "lb-http" {
  source            = "GoogleCloudPlatform/lb-http/google//modules/serverless_negs"
  version           = "~> 6.2.0"

  project           = var.project_id
  name              = "${var.prefix}-load-balancer"

  managed_ssl_certificate_domains = ["gcp.fleetdm.com"]
  ssl                             = true
  https_redirect                  = true

  backends = {
    default = {
      # List your serverless NEGs, VMs, or buckets as backends
      groups = [
        {
          group = google_compute_region_network_endpoint_group.neg.id
        }
      ]
      custom_request_headers  = null
      custom_response_headers = null

      enable_cdn = false

      log_config = {
        enable      = true
        sample_rate = 1.0
      }

      iap_config = {
        enable               = false
        oauth2_client_id     = null
        oauth2_client_secret = null
      }

      description             = null
      custom_request_headers  = null
      security_policy         = null
    }
  }
}
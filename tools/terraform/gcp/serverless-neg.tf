module "lb-http" {
  source            = "GoogleCloudPlatform/lb-http/google//modules/serverless_negs"
  version           = "~> 6.2.0"

  project           = "YOUR_PROJECT_ID"
  name              = "my-lb"

  managed_ssl_certificate_domains = ["YOUR_DOMAIN.COM"]
  ssl                             = true
  https_redirect                  = true

  backends = {
    default = {
      # List your serverless NEGs, VMs, or buckets as backends
      groups = [
        {
          group = google_compute_region_network_endpoint_group.default.id
        }
      ]

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
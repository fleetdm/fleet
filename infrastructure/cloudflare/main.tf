terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
  backend "s3" {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "infrastructure/cloudflare/terraform.tfstate"
    workspace_key_prefix = "infrastructure"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
  }
}

provider "cloudflare" {
  # API token provided via CLOUDFLARE_API_TOKEN environment variable
}

data "cloudflare_zone" "fleetdm_com" {
  name = "fleetdm.com"
}

# Salesforce email domain verification TXT record
resource "cloudflare_record" "salesforce_email_domain_verification" {
  zone_id = data.cloudflare_zone.fleetdm_com.id
  name    = "fleetdm.com"
  type    = "TXT"
  content = "00D4x000005QgaP=1TBUG00000003KT"
  ttl     = 1 # 1 = automatic
  comment = "Salesforce email domain verification"
}

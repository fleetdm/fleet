# MDM
resource "tls_private_key" "scep_key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "tls_self_signed_cert" "scep_cert" {
  private_key_pem = tls_private_key.scep_key.private_key_pem

  subject {
    common_name  = "Fleet Root CA"
    organization = "Fleet."
    country      = "US"
  }

  is_ca_certificate     = true
  validity_period_hours = 87648

  allowed_uses = [
    "cert_signing",
    "crl_signing",
    "key_encipherment",
    "digital_signature",
  ]
}

# Cloudfront
resource "tls_private_key" "cloudfront_key" {
  algorithm = "RSA"
  rsa_bits  = 2048
}
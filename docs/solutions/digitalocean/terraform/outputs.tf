output "fleet_url" {
  description = "Open this address to finish setting up Fleet."
  value       = var.domain != "" ? "https://${var.domain}" : digitalocean_app.fleet.live_url
}

output "dns_target" {
  description = "If you set a custom domain, create a CNAME record that points at this host name."
  value       = replace(digitalocean_app.fleet.default_ingress, "https://", "")
}

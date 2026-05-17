output "pmm_url" {
  description = "URL to access PMM UI (internal only, requires VPN)"
  value       = "http://pmm.${terraform.workspace}.loadtest.fleetdm.com"
}

output "pmm_admin_password_secret" {
  description = "Secrets Manager secret name for PMM admin password"
  value       = aws_secretsmanager_secret.pmm_admin_password.name
}

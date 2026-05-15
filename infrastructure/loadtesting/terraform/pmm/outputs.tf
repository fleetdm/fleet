output "pmm_private_ip" {
  description = "Private IP of the PMM server instance"
  value       = aws_instance.pmm.private_ip
}

output "pmm_instance_id" {
  description = "EC2 instance ID of the PMM server"
  value       = aws_instance.pmm.id
}

output "pmm_url" {
  description = "URL to access PMM UI (internal only, requires VPN)"
  value       = "https://pmm.${terraform.workspace}.loadtest.fleetdm.com"
}

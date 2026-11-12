output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_certificate_authority_data" {
  description = "EKS cluster CA certificate"
  value       = module.eks.cluster_certificate_authority_data
}

output "configure_kubectl" {
  value = "aws eks update-kubeconfig --region ${data.aws_region.current.region} --name ${module.eks.cluster_name}"
}

output "signoz_ui_url" {
  value = "https://${local.signoz_domain}"
}

# Output for programmatic access - internal LoadBalancer hostname
output "otel_collector_endpoint" {
  description = "Internal OTLP collector endpoint (https://host:port)"
  value       = "https://${local.otlp_domain}:443"
}

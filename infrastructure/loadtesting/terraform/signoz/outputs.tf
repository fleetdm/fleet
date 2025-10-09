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
  value = "aws eks update-kubeconfig --region ${var.aws_region} --name ${module.eks.cluster_name}"
}

output "get_signoz_ui_url" {
  value = "kubectl get svc -n signoz signoz -o jsonpath='{.status.loadBalancer.ingress[0].hostname}':8080"
}

output "get_otlp_endpoint" {
  value = "kubectl get svc -n signoz signoz-otel-collector -o jsonpath='{.status.loadBalancer.ingress[0].hostname}':4317"
}

# Data source to get the OTLP collector service
data "kubernetes_service" "otlp_collector" {
  metadata {
    name      = "signoz-otel-collector"
    namespace = "signoz"
  }

  depends_on = [helm_release.signoz]
}

# Output for programmatic access - internal LoadBalancer hostname
output "otel_collector_endpoint" {
  description = "Internal OTLP collector endpoint (hostname:port)"
  value       = "${data.kubernetes_service.otlp_collector.status[0].load_balancer[0].ingress[0].hostname}:4317"
}

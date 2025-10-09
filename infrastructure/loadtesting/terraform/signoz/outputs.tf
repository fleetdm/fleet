output "cluster_name" {
  value = module.eks.cluster_name
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

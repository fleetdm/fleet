# SigNoz deployment for OpenTelemetry tracing
# Conditionally deployed when var.enable_otel = true

module "signoz" {
  count  = var.enable_otel ? 1 : 0
  source = "../signoz"

  aws_region   = data.aws_region.current.region
  cluster_name = "signoz-${terraform.workspace}"

  # Use dedicated EKS VPC with proper Kubernetes tags
  vpc_id     = data.terraform_remote_state.eks_vpc[0].outputs.vpc.vpc_id
  subnet_ids = data.terraform_remote_state.eks_vpc[0].outputs.vpc.private_subnets

  providers = {
    aws        = aws
    helm       = helm.signoz
    kubernetes = kubernetes.signoz
  }
}

# Outputs from SigNoz module
output "signoz_cluster_name" {
  description = "SigNoz EKS cluster name"
  value       = var.enable_otel ? module.signoz[0].cluster_name : null
}

output "signoz_otel_collector_endpoint" {
  description = "Internal OTLP collector endpoint for Fleet"
  value       = var.enable_otel ? module.signoz[0].otel_collector_endpoint : null
}

output "signoz_configure_kubectl" {
  description = "Command to configure kubectl for SigNoz"
  value       = var.enable_otel ? module.signoz[0].configure_kubectl : null
}

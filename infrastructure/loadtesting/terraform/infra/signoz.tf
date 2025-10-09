# SigNoz deployment for OpenTelemetry tracing
# Conditionally deployed when var.enable_otel = true

module "signoz" {
  count  = var.enable_otel ? 1 : 0
  source = "../signoz"

  aws_region   = "us-east-2"
  cluster_name = "signoz-${terraform.workspace}"

  # Use shared fleet VPC
  vpc_id     = data.terraform_remote_state.shared.outputs.vpc.vpc_id
  subnet_ids = data.terraform_remote_state.shared.outputs.vpc.private_subnets

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

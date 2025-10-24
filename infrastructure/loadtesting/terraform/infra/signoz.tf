# SigNoz deployment for OpenTelemetry tracing
# SigNoz is deployed as a separate Terraform root module
# This reads its outputs via remote state

# Read SigNoz deployment from remote state
data "terraform_remote_state" "signoz" {
  count   = var.enable_otel ? 1 : 0
  backend = "s3"
  config = {
    bucket               = "fleet-terraform-state20220408141538466600000002"
    key                  = "loadtesting/loadtesting/signoz/terraform.tfstate"
    workspace_key_prefix = "loadtesting"
    region               = "us-east-2"
    encrypt              = true
    kms_key_id           = "9f98a443-ffd7-4dbe-a9c3-37df89b2e42a"
    dynamodb_table       = "tf-remote-state-lock"
    assume_role = {
      role_arn = "arn:aws:iam::353365949058:role/terraform-loadtesting"
    }
  }
  workspace = terraform.workspace
}

# Outputs from SigNoz remote state
output "signoz_cluster_name" {
  description = "SigNoz EKS cluster name"
  value       = var.enable_otel ? try(data.terraform_remote_state.signoz[0].outputs.cluster_name, "SigNoz not deployed yet") : null
}

output "signoz_otel_collector_endpoint" {
  description = "Internal OTLP collector endpoint for Fleet"
  value       = var.enable_otel ? try(data.terraform_remote_state.signoz[0].outputs.otel_collector_endpoint, "SigNoz not deployed yet") : null
}

output "signoz_configure_kubectl" {
  description = "Command to configure kubectl for SigNoz"
  value       = var.enable_otel ? try(data.terraform_remote_state.signoz[0].outputs.configure_kubectl, "SigNoz not deployed yet") : null
}

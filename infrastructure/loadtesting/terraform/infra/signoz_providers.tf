# Provider configurations for SigNoz module
# These must be in a separate file to avoid circular dependencies

# Note: These providers will only be used when enable_otel = true and the SigNoz module is instantiated
# The provider configuration block itself always exists but is only actively used when the module calls it

provider "helm" {
  alias = "signoz"

  # Configuration is dynamic based on whether SigNoz is enabled
  # When enable_otel = false, these values are empty strings/lists but the provider block must exist
  kubernetes {
    host                   = try(module.signoz[0].cluster_endpoint, "")
    cluster_ca_certificate = try(base64decode(module.signoz[0].cluster_certificate_authority_data), "")

    exec {
      api_version = "client.authentication.k8s.io/v1beta1"
      command     = "aws"
      args        = try(["eks", "get-token", "--cluster-name", module.signoz[0].cluster_name], [])
    }
  }
}

provider "kubernetes" {
  alias = "signoz"

  host                   = try(module.signoz[0].cluster_endpoint, "")
  cluster_ca_certificate = try(base64decode(module.signoz[0].cluster_certificate_authority_data), "")

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = try(["eks", "get-token", "--cluster-name", module.signoz[0].cluster_name], [])
  }
}

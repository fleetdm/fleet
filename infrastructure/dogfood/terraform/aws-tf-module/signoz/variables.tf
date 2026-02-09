variable "otel_bearer_token" {
  type        = string
  sensitive   = true
  description = "Bearer token required by the SigNoz OTLP collector."
}

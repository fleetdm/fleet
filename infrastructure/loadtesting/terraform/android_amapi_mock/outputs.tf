output "mock_url" {
  description = "Internal URL for the Android AMAPI mock (use as FLEET_DEV_ANDROID_PROXY_ENDPOINT and --android_proxy_address)"
  value       = "http://${data.terraform_remote_state.infra.outputs.internal_alb_dns_name}"
}

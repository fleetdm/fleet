locals {
  name            = "fleetdm"
  prefix          = "fleet"
  domain_fleetdm  = "loadtest.fleetdm.com"
  domain_fleetctl = "loadtest.fleetctl.com"
  additional_env_vars = [for k, v in merge({
    "FLEET_VULNERABILITIES_DATABASES_PATH" : "/home/fleet"
    "FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING" : "false"
    "FLEET_LOGGING_DEBUG" : "true"
    //"OTEL_EXPORTER_OTLP_ENDPOINT" : "http://10.10.2.193:8200"
    //"FLEET_LOGGING_TRACING_ENABLED" : "true"
    "ELASTIC_APM_SERVER_URL" : "http://10.10.2.193:8200"
    "ELASTIC_APM_SERVICE_NAME" : "fleet"
    "ELASTIC_APM_ENVIRONMENT" : "loadtest"
    "ELASTIC_APM_TRANSACTION_SAMPLE_RATE" : "0.004"
    "ELASTIC_APM_SERVICE_VERSION" : "${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  }, var.fleet_config) : { name = k, value = v }]
}

locals {
  name            = "fleetdm"
  prefix          = "fleet"
  domain_fleetdm  = "loadtest.fleetdm.com"
  domain_fleetctl = "loadtest.fleetctl.com"
  additional_env_vars = [for k, v in merge({
    "FLEET_VULNERABILITIES_DATABASES_PATH" : "/home/fleet"
    "FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING" : "false"
    "FLEET_LOGGING_DEBUG" : "true"
  }, var.fleet_config) : { name = k, value = v }]
}

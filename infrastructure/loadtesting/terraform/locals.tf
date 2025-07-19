locals {
  name   = "fleetdm-${terraform.workspace}"
  prefix = "fleet-${terraform.workspace}"
  extra_execution_iam_policies = concat(
    module.cloudfront-software-installers.extra_execution_iam_policies,
  )
  iam = {
    role = {
      name        = "${local.customer}-role"
      policy_name = "${local.customer}-iam-policy"
    }
    execution = {
      name        = "${local.customer}-execution-role"
      policy_name = "${local.customer}-iam-policy-execution"
    }
  }
  additional_env_vars = [for k, v in merge({
    "FLEET_VULNERABILITIES_DATABASES_PATH" : "/home/fleet"
    "FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING" : "false"
    "FLEET_LOGGING_DEBUG" : "true"
    "FLEET_LOGGING_TRACING_ENABLED" : "true"
    "FLEET_LOGGING_TRACING_TYPE" : "elasticapm"
    "ELASTIC_APM_SERVER_URL" : "https://loadtest.fleetdm.com:8200"
    "ELASTIC_APM_SERVICE_NAME" : "fleet"
    "ELASTIC_APM_ENVIRONMENT" : "${terraform.workspace}"
    "ELASTIC_APM_TRANSACTION_SAMPLE_RATE" : "0.004"
    "ELASTIC_APM_SERVICE_VERSION" : "${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  }, var.fleet_config) : { name = k, value = v }]
  # Private Subnets from VPN VPC
  vpn_cidr_blocks = [
    "10.255.1.0/24",
    "10.255.2.0/24",
    "10.255.3.0/24",
  ]
  software_installers_kms_policy = [{
    sid = "AllowSoftwareInstallersKMSAccess"
    actions = [
      "kms:ReEncrypt*",
      "kms:GenerateDataKey*",
      "kms:Encrypt*",
      "kms:Describe*",
      "kms:Decrypt*"
    ]
    resources = [aws_kms_key.software_installers.arn]
    effect    = "Allow"
  }]
  extra_secrets = merge(
    # module.cloudfront-software-installers.extra_secrets
  )
  secrets = [for k, v in local.extra_secrets : {
    name      = k
    valueFrom = v
  }]
  cloudfront_key_basename = "cloudfront"
}

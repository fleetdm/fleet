locals {
  name   = "fleetdm-${terraform.workspace}"
  prefix = "fleet-${terraform.workspace}"
  extra_execution_iam_policies = concat(
    # module.cloudfront-software-installers.extra_execution_iam_policies,
    []
  )
  iam = {
    role = {
      name        = "${terraform.workspace}-role"
      policy_name = "${terraform.workspace}-iam-policy"
    }
    execution = {
      name        = "${terraform.workspace}-execution-role"
      policy_name = "${terraform.workspace}-iam-policy-execution"
    }
  }
  additional_env_vars = [for k, v in merge({
    "FLEET_VULNERABILITIES_DATABASES_PATH" : "/home/fleet"
    "FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING" : "false"
    "FLEET_LOGGING_DEBUG" : "true"
    # No Elastic APM: its instrumentation is gorilla-specific, so the server turns off its stdlib ServeMux fast path
    # whenever it is active. Tracing can be added per-run through var.fleet_config.
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
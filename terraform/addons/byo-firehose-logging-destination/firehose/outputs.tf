output "fleet_extra_environment_variables" {
  value = {
    FLEET_FIREHOSE_STATUS_STREAM       = var.firehose_status_name
    FLEET_FIREHOSE_RESULT_STREAM       = var.firehose_results_name
    FLEET_FIREHOSE_AUDIT_STREAM        = var.firehose_audit_name
    FLEET_FIREHOSE_STS_ASSUME_ROLE_ARN = var.iam_role_arn
    FLEET_FIREHOSE_REGION              = var.region
    FLEET_OSQUERY_STATUS_LOG_PLUGIN    = "firehose"
    FLEET_OSQUERY_RESULT_LOG_PLUGIN    = "firehose"
    FLEET_ACTIVITY_ENABLE_AUDIT_LOG    = length(var.firehose_audit_name) > 0 ? "true" : "false"
    FLEET_ACTIVITY_AUDIT_LOG_PLUGIN    = "firehose" # only has an effect if ^ is true
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.fleet-assume-role.arn
  ]
}

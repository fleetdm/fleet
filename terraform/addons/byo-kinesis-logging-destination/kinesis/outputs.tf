output "fleet_extra_environment_variables" {
  value = {
    FLEET_KINESIS_STATUS_STREAM       = var.kinesis_status_name
    FLEET_KINESIS_RESULT_STREAM       = var.kinesis_results_name
    FLEET_KINESIS_AUDIT_STREAM        = var.kinesis_audit_name
    FLEET_KINESIS_STS_ASSUME_ROLE_ARN = var.iam_role_arn
    FLEET_KINESIS_STS_EXTERNAL_ID     = var.sts_external_id
    FLEET_KINESIS_REGION              = var.region
    FLEET_OSQUERY_STATUS_LOG_PLUGIN   = length(var.kinesis_status_name) > 0 ? "kinesis" : ""
    FLEET_OSQUERY_RESULT_LOG_PLUGIN   = length(var.kinesis_results_name) > 0 ? "kinesis" : ""
    FLEET_ACTIVITY_AUDIT_LOG_PLUGIN   = length(var.kinesis_audit_name) > 0 ? "kinesis" : ""
    FLEET_ACTIVITY_ENABLE_AUDIT_LOG   = length(var.kinesis_audit_name) > 0 ? "true" : "false"
  }
}

output "fleet_extra_iam_policies" {
  value = [
    aws_iam_policy.fleet-assume-role.arn
  ]
}

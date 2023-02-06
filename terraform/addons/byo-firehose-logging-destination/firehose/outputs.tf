data "aws_region" "current" {}
output "fleet_extra_environment_variables" {
  value = {
    FLEET_FIREHOSE_STATUS_STREAM       = var.firehose_status_name
    FLEET_FIREHOSE_RESULT_STREAM       = var.firehose_results_name
    FLEET_FIREHOSE_STS_ASSUME_ROLE_ARN = var.iam_role_arn
    FLEET_FIREHOSE_REGION              = data.aws_region.current.name
    FLEET_OSQUERY_STATUS_LOG_PLUGIN    = "firehose"
    FLEET_OSQUERY_RESULT_LOG_PLUGIN    = "firehose"
  }
}
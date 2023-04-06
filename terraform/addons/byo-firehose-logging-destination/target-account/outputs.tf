output "firehose_iam_role" {
  value = aws_iam_role.fleet_role.arn
}

output "s3_destination" {
  value = aws_s3_bucket.destination.arn
}

output "firehose_results" {
  value = aws_kinesis_firehose_delivery_stream.osquery_results.name
}

output "firehose_status" {
  value = aws_kinesis_firehose_delivery_stream.osquery_status.name
}
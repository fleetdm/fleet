output "firehose_iam_role" {
  value = aws_iam_role.fleet_role.arn
}

output "s3_destination" {
  value = aws_s3_bucket.destination.arn
}

output "log_destinations" {
  description = "Map of Firehose delivery streams' names."
  value = { for key, stream in aws_kinesis_firehose_delivery_stream.fleet_log_destinations : key => stream.name }
}

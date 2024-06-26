output "kinesis_iam_role" {
  value = aws_iam_role.fleet_role.arn
}

output "kinesis_streams" {
  description = "A map of Kinesis streams with their names and ARNs."
  value = {
    for k, stream in aws_kinesis_stream.fleet_log_destination : k => {
      stream_name = stream.name
      stream_arn  = stream.arn
    }
  }
}

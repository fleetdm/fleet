resource "aws_kms_key" "kinesis_key" {
  description = "KMS key for encrypting Kinesis Data Streams for Fleet logging destinations."
}

resource "aws_kinesis_stream" "fleet_log_destination" {
  for_each            = var.log_destinations
  name                = each.value.name
  encryption_type     = "KMS"
  kms_key_id          = aws_kms_key.kinesis_key.id
  shard_level_metrics = each.value.shard_level_metrics
  shard_count         = each.value.stream_mode == "ON_DEMAND" ? null : each.value.shard_count
  stream_mode_details {
    stream_mode = each.value.stream_mode
  }
}

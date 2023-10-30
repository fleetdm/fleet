output "cron_monitoring_security_group_id" {
  value = try(aws_security_group.cron_monitoring[0].id, null)
}

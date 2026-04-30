import {
  to = module.monitoring.aws_cloudwatch_log_group.cron_monitoring_lambda[0]
  id = "/aws/lambda/${local.customer}_cron_monitoring"
}

import {
  to = module.logging_alb.aws_glue_catalog_table.partitioned_alb_logs[0]
  id = "160035666661:fleet_dogfood_alb_logs:partitioned_alb_logs"
}
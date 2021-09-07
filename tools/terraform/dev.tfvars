scaling_configuration = {
  auto_pause               = true
  min_capacity             = 2
  max_capacity             = 16
  seconds_until_auto_pause = 300
  timeout_action           = "ForceApplyCapacityChange"
}

cert_arn = "arn:aws:acm:us-east-2:160035666661:certificate/9b810820-c589-4fdb-8d32-be8d8d0f0c89"
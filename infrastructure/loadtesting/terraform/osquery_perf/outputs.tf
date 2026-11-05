output "osquery_perf" {
  value = {
    module          = module.osquery_perf
    stable_services = [for s in aws_ecs_service.osquery_perf_stable : s.name]
  }
}

output "alb" {
  value = module.alb
}

output "ecs_service_name" {
  value = aws_ecs_service.mdmproxy.name
}

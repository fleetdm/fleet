output "alb_security_group" {
  value = aws_security_group.lb
}

output "alb" {
  value = aws_alb.main
}

output "alb-listener" {
  value = aws_alb_listener.https-fleetdm
}

output "vpc" {
  value = module.vpc
}

output "ecr" {
  value = aws_ecr_repository.prometheus-to-cloudwatch
}

output "ecr-kms" {
  value = aws_kms_key.main
}

output "enroll_secret" {
  value = aws_secretsmanager_secret.enroll_secret
}

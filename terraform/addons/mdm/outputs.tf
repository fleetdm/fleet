output "extra_environment_variables" {
  value = merge({
    FLEET_MDM_APPLE_SERVER_ADDRESS = var.public_domain_name
    }, var.enable_windows_mdm == false ? {} : {
    FLEET_DEV_MDM_ENABLED = "1"
  })
}

output "extra_secrets" {
  value = merge({
    FLEET_MDM_APPLE_SCEP_CERT_BYTES   = "${aws_secretsmanager_secret.scep.arn}:crt::"
    FLEET_MDM_APPLE_SCEP_CA_CERT_PEM  = "${aws_secretsmanager_secret.scep.arn}:crt::"
    FLEET_MDM_APPLE_SCEP_KEY_BYTES    = "${aws_secretsmanager_secret.scep.arn}:key::"
    FLEET_MDM_APPLE_SCEP_CA_KEY_PEM   = "${aws_secretsmanager_secret.scep.arn}:key::"
    FLEET_MDM_APPLE_SCEP_CHALLENGE    = "${aws_secretsmanager_secret.scep.arn}:challenge::"
    FLEET_MDM_APPLE_APNS_CERT_BYTES   = "${aws_secretsmanager_secret.apn.arn}:FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM::"
    FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM = "${aws_secretsmanager_secret.apn.arn}:FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM::"
    FLEET_MDM_APPLE_APNS_KEY_BYTES    = "${aws_secretsmanager_secret.apn.arn}:FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM::"
    FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM  = "${aws_secretsmanager_secret.apn.arn}:FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM::"
    }, var.dep_secret_name == null ? {} : {
    FLEET_MDM_APPLE_DEP_TOKEN             = "${aws_secretsmanager_secret.dep[0].arn}:token::"
    FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES = "${aws_secretsmanager_secret.dep[0].arn}:token-encrypted::"
    FLEET_MDM_APPLE_BM_CERT_BYTES         = "${aws_secretsmanager_secret.dep[0].arn}:cert::"
    FLEET_MDM_APPLE_BM_KEY_BYTES          = "${aws_secretsmanager_secret.dep[0].arn}:key::"
    }, var.enable_windows_mdm == false ? {} : {
    FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT = "${aws_secretsmanager_secret.scep.arn}:crt::"
    FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY  = "${aws_secretsmanager_secret.scep.arn}:key::"
  })
}

output "extra_execution_iam_policies" {
  value = [
    aws_iam_policy.main.arn
  ]
}

output "scep" {
  value = aws_secretsmanager_secret.scep
}

output "dep" {
  value = var.dep_secret_name == null ? null : aws_secretsmanager_secret.dep[0]
}

output "apn" {
  value = aws_secretsmanager_secret.apn
}

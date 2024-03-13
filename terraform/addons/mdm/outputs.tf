output "extra_secrets" {
  value = merge(var.enable_apple_mdm == false ? {} : {
    FLEET_MDM_APPLE_SCEP_CERT_BYTES   = "${aws_secretsmanager_secret.scep.arn}:FLEET_MDM_APPLE_SCEP_CERT_BYTES::"
    FLEET_MDM_APPLE_SCEP_KEY_BYTES    = "${aws_secretsmanager_secret.scep.arn}:FLEET_MDM_APPLE_SCEP_KEY_BYTES::"
    FLEET_MDM_APPLE_SCEP_CHALLENGE    = "${aws_secretsmanager_secret.scep.arn}:FLEET_MDM_APPLE_SCEP_CHALLENGE::"
    FLEET_MDM_APPLE_APNS_CERT_BYTES   = "${aws_secretsmanager_secret.apn[0].arn}:FLEET_MDM_APPLE_APNS_CERT_BYTES::"
    FLEET_MDM_APPLE_APNS_KEY_BYTES    = "${aws_secretsmanager_secret.apn[0].arn}:FLEET_MDM_APPLE_APNS_KEY_BYTES::"
    }, var.abm_secret_name == null || var.enable_apple_mdm == false ? {} : {
    FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES = "${aws_secretsmanager_secret.abm[0].arn}:FLEET_MDM_APPLE_BM_SERVER_TOKEN_BYTES::"
    FLEET_MDM_APPLE_BM_CERT_BYTES         = "${aws_secretsmanager_secret.abm[0].arn}:FLEET_MDM_APPLE_BM_CERT_BYTES::"
    FLEET_MDM_APPLE_BM_KEY_BYTES          = "${aws_secretsmanager_secret.abm[0].arn}:FLEET_MDM_APPLE_BM_KEY_BYTES::"
    }, var.enable_windows_mdm == false ? {} : {
    FLEET_MDM_WINDOWS_WSTEP_IDENTITY_CERT_BYTES = "${aws_secretsmanager_secret.scep.arn}:FLEET_MDM_APPLE_SCEP_CERT_BYTES::"
    FLEET_MDM_WINDOWS_WSTEP_IDENTITY_KEY_BYTES  = "${aws_secretsmanager_secret.scep.arn}:FLEET_MDM_APPLE_SCEP_KEY_BYTES::"
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

output "abm" {
  value = var.abm_secret_name == null ? null : aws_secretsmanager_secret.abm[0]
}

output "apn" {
  value = var.enable_apple_mdm == false ? null : aws_secretsmanager_secret.apn[0]
}

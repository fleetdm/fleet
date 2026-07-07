resource "aws_secretsmanager_secret" "enroll_secret" {
  name       = "/fleet/loadtest/enroll/${random_pet.main.id}"
  kms_key_id = aws_kms_key.main.id
}

# Google service account credentials for Android AMAPI mock forwarding.
resource "aws_secretsmanager_secret" "android_google_credentials" {
  name       = "/fleet/loadtest/android-google-credentials"
  kms_key_id = aws_kms_key.main.id
}

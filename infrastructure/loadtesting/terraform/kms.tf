resource "aws_kms_key" "customer_data_key" {
  description = "key used to encrypt sensitive data stored in terraform"
}

resource "aws_kms_alias" "alias" {
  name          = "alias/${terraform.workspace}-terraform-encrypted"
  target_key_id = aws_kms_key.customer_data_key.id
}

output "kms_key_id" {
  value = aws_kms_key.customer_data_key.id
}
module "cloudfront-software-installers" {
  source            = "github.com/fleetdm/fleet-terraform/addons/cloudfront-software-installers?ref=tf-mod-addon-cloudfront-software-installers-v1.0.1"
  customer          = terraform.workspace
  s3_bucket         = module.loadtest.byo-db.byo-ecs.fleet_s3_software_installers_config.bucket_name
  s3_kms_key_id     = module.loadtest.byo-db.byo-ecs.fleet_s3_software_installers_config.kms_key_id
  public_key        = tls_private_key.cloudfront_key.public_key_pem
  private_key       = tls_private_key.cloudfront_key.private_key_pem
  enable_logging    = true
  logging_s3_bucket = module.logging_alb.log_s3_bucket_id
}

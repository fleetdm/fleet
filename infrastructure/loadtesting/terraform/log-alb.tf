module "logging_alb" {
  source        = "github.com/fleetdm/fleet-terraform/addons/logging-alb?depth=1&ref=tf-mod-addon-logging-alb-v1.3.0"
  prefix        = "${terraform.workspace}"
  enable_athena = true
}
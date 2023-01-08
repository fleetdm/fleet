module "main" {
  source          = "../"
  certificate_arn = module.acm.acm_certificate_arn
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "fleet.loadtest.fleetdm.com"
  zone_id     = data.aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_record" "main" {
  zone_id = data.aws_route53_zone.main.id
  name    = "fleet.loadtest.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.main.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.main.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}

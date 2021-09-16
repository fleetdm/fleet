resource "aws_route53_zone" "dogfood_fleetctl_com" {
  name = "dogfood.fleetctl.com"
}

resource "aws_route53_record" "dogfood_fleetctl_com" {
  zone_id = aws_route53_zone.dogfood_fleetctl_com.zone_id
  name    = "dogfood.fleetctl.com"
  type    = "A"

  alias {
    name                   = aws_alb.main.dns_name
    zone_id                = aws_alb.main.zone_id
    evaluate_target_health = false
  }
}

resource "aws_acm_certificate" "dogfood_fleetctl_com" {
  domain_name       = "dogfood.fleetctl.com"
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "dogfood_fleetctl_com_validation" {
  for_each = {
    for dvo in aws_acm_certificate.dogfood_fleetctl_com.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }

  allow_overwrite = true
  name            = each.value.name
  records         = [each.value.record]
  ttl             = 60
  type            = each.value.type
  zone_id         = aws_route53_zone.dogfood_fleetctl_com.zone_id
}

resource "aws_acm_certificate_validation" "dogfood_fleetctl_com" {
  certificate_arn         = aws_acm_certificate.dogfood_fleetctl_com.arn
  validation_record_fqdns = [for record in aws_route53_record.dogfood_fleetctl_com_validation : record.fqdn]
}
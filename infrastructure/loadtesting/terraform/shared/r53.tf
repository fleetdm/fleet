resource "aws_route53_zone" "fleetctl_com" {
  name = "loadtest.fleetctl.com"
}

resource "aws_route53_zone" "fleetdm_com" {
  name = "loadtest.fleetdm.com"
}

resource "aws_route53_record" "fleetctl_com" {
  zone_id = aws_route53_zone.fleetctl_com.zone_id
  name    = aws_route53_zone.fleetctl_com.name
  type    = "A"

  alias {
    name                   = aws_alb.main.dns_name
    zone_id                = aws_alb.main.zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "fleetdm_com" {
  zone_id = aws_route53_zone.fleetdm_com.zone_id
  name    = aws_route53_zone.fleetdm_com.name
  type    = "A"

  alias {
    name                   = aws_alb.main.dns_name
    zone_id                = aws_alb.main.zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "wildcard" {
  zone_id = aws_route53_zone.fleetdm_com.zone_id
  name    = "*.${aws_route53_zone.fleetdm_com.name}"
  type    = "A"

  alias {
    name                   = aws_alb.main.dns_name
    zone_id                = aws_alb.main.zone_id
    evaluate_target_health = false
  }
}

resource "aws_acm_certificate" "wildcard" {
  domain_name               = aws_route53_record.wildcard.name
  subject_alternative_names = [aws_route53_record.fleetdm_com.name]
  validation_method         = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "wildcard_validation" {
  for_each = {
    for dvo in aws_acm_certificate.wildcard.domain_validation_options : dvo.domain_name => {
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
  zone_id         = aws_route53_zone.fleetdm_com.zone_id
}

resource "aws_acm_certificate_validation" "wildcard" {
  certificate_arn         = aws_acm_certificate.wildcard.arn
  validation_record_fqdns = [for record in aws_route53_record.wildcard_validation : record.fqdn]
}

resource "aws_acm_certificate" "fleetdm_com" {
  domain_name       = aws_route53_record.fleetdm_com.name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "fleetdm_com_validation" {
  for_each = {
    for dvo in aws_acm_certificate.fleetdm_com.domain_validation_options : dvo.domain_name => {
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
  zone_id         = aws_route53_zone.fleetdm_com.zone_id
}

resource "aws_acm_certificate_validation" "fleetdm_com" {
  certificate_arn         = aws_acm_certificate.fleetdm_com.arn
  validation_record_fqdns = [for record in aws_route53_record.fleetdm_com_validation : record.fqdn]
}

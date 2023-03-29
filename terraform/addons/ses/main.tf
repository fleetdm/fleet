data "aws_route53_zone" "main" {
  name = var.domain
}

module "labels" {
  source  = "clouddrove/labels/aws"
  version = "1.3.0"

  name        = var.name
  environment = var.environment
  managedby   = var.managedby
  label_order = var.label_order
  repository  = var.repository
}


resource "aws_ses_domain_identity" "default" {
  domain = var.domain
}

###DNS VERIFICATION#######

resource "aws_ses_domain_identity_verification" "default" {
  domain     = aws_ses_domain_identity.default.id
  depends_on = [aws_route53_record.ses_verification]
}

resource "aws_route53_record" "ses_verification" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = module.labels.id
  type    = var.txt_type
  ttl     = "600"
  records = [aws_ses_domain_identity.default.verification_token]
}

resource "aws_ses_domain_dkim" "default" {
  domain = aws_ses_domain_identity.default.domain
}

###DKIM VERIFICATION#######

resource "aws_route53_record" "dkim" {
  for_each = aws_ses_domain_dkim.default.dkim_tokens
  zone_id  = data.aws_route53_zone.main.zone_id
  name     = format("%s._domainkey.%s", each.key, var.domain)
  type     = var.cname_type
  ttl      = 600
  records  = [format("%s.dkim.amazonses.com", each.key)]
}

resource "aws_route53_record" "spf_domain" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = module.labels.id
  type    = var.txt_type
  ttl     = "600"
  records = ["v=spf1 include:amazonses.com -all"]
}

resource "aws_iam_policy" "main" {
  policy = data.aws_iam_policy_document.main.json
}

data "aws_iam_policy_document" "main" {
  statement {
    actions = [
      "ses:SendEmail",
      "ses:SendRawEmail",
    ]
    resources = ["*"]
    condition {
      test     = "StringLike"
      variable = "ses:FromAddress"
      values = [
        "*@${var.domain}"
      ]
    }
  }
}

data "aws_route53_zone" "main" {
  name = var.domain
}

resource "aws_ses_domain_identity" "default" {
  domain = var.domain
}

resource "aws_ses_domain_dkim" "default" {
  domain = aws_ses_domain_identity.default.domain
}

###DKIM VERIFICATION#######

resource "aws_route53_record" "dkim" {
  for_each = toset(aws_ses_domain_dkim.default.dkim_tokens)
  zone_id  = data.aws_route53_zone.main.zone_id
  name     = format("%s._domainkey.%s", each.key, var.domain)
  type     = "CNAME"
  ttl      = 600
  records  = [format("%s.dkim.amazonses.com", each.key)]
}

resource "aws_route53_record" "spf_domain" {
  zone_id = data.aws_route53_zone.main.zone_id
  name    = ""
  type    = "TXT"
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

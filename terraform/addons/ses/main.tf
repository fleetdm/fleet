locals {
  spf_domains = [
    aws_ses_domain_identity.default.domain,
    "_amazonses.${aws_ses_domain_identity.default.domain}"
  ]
}

resource "aws_ses_domain_identity" "default" {
  domain = var.domain
}

resource "aws_ses_domain_dkim" "default" {
  domain = aws_ses_domain_identity.default.domain
}

###DKIM VERIFICATION#######

resource "aws_route53_record" "amazonses_dkim_record" {
  count   = 3 // no clue why this is three, but multiple modules all did the same thing
  zone_id = var.zone_id
  name    = "${element(aws_ses_domain_dkim.default.dkim_tokens, count.index)}._domainkey.${var.domain}"
  type    = "CNAME"
  ttl     = "600"
  records = ["${element(aws_ses_domain_dkim.default.dkim_tokens, count.index)}.dkim.amazonses.com"]
}


resource "aws_route53_record" "spf_domain" {
  for_each = toset(local.spf_domains)
  zone_id  = var.zone_id
  name     = each.key
  type     = "TXT"
  ttl      = "600"
  records  = each.key == aws_ses_domain_identity.default.domain ? flatten([["v=spf1 include:amazonses.com -all"], var.extra_txt_records]) : ["v=spf1 include:amazonses.com -all"]
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

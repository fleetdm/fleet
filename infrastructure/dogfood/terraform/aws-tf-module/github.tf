data "tls_certificate" "github" {
  url = "https://token.actions.githubusercontent.com/.well-known/openid-configuration"
}

/*
It's possible to use the following to add Github as an OpenID Connect Provider and integrate
Github Actions as your CI/CD mechanism.
*/

resource "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"

  client_id_list = [
    "sts.amazonaws.com",
  ]


  thumbprint_list = [
    data.tls_certificate.github.certificates[0].sha1_fingerprint
  ]
}

resource "aws_iam_role" "gha_role" {
  name               = "github-actions-role"
  assume_role_policy = data.aws_iam_policy_document.gha_assume_role.json
}

resource "aws_iam_role_policy" "gha_role_policy" {
  policy = data.aws_iam_policy_document.gha-permissions.json
  role   = aws_iam_role.gha_role.id
}


#####################
# AssumeRole
#
# Allow sts:AssumeRoleWithWebIdentity from GitHub via OIDC
# Customize your repository
#####################
data "aws_iam_policy_document" "gha_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type = "Federated"
      identifiers = [
        "arn:aws:iam::${data.aws_caller_identity.current.account_id}:oidc-provider/token.actions.githubusercontent.com"
      ]
    }
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:fleetdm/fleet:*"]
    }

    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
  }
}
// Customize the permissions for your deployment
data "aws_iam_policy_document" "gha-permissions" {
  statement {
    effect = "Allow"
    actions = [
      "ec2:*",
      "cloudwatch:*",
      "s3:*",
      "lambda:*",
      "ecs:*",
      "rds:*",
      "rds-data:*",
      "secretsmanager:*",
      "pi:*",
      "ecr:*",
      "iam:*",
      "aps:*",
      "vpc:*",
      "kms:*",
      "elasticloadbalancing:*",
      "ce:*",
      "cur:*",
      "logs:*",
      "cloudformation:*",
      "ssm:*",
      "sns:*",
      "elasticache:*",
      "application-autoscaling:*",
      "acm:*",
      "route53:*",
      "dynamodb:*",
      "kinesis:*",
      "firehose:*",
      "athena:*",
      "glue:*",
      "ses:*",
      "wafv2:*",
      "events:*",
      "cloudfront:*",
      "backup:*",
      "backup-storage:*"
    ]
    resources = ["*"]
  }
}

# --- Dynamically fetch GitHub OIDC TLS cert ---
data "tls_certificate" "github" {
  url = "https://token.actions.githubusercontent.com/.well-known/openid-configuration"
}

/*
It's possible to use the following to add GitHub as an OpenID Connect Provider and integrate
GitHub Actions as your CI/CD mechanism.
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

# --- Trust policy for fleetdm/confidential repo ---
data "aws_iam_policy_document" "fleetdm_confidential_cloudflare_trust" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]

    principals {
      type        = "Federated"
      identifiers = [aws_iam_openid_connect_provider.github.arn]
    }

    # Required audience condition to ensure only AWS STS can use this role
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }

    # Restrict to the fleetdm/confidential repo only
    condition {
      test     = "StringLike"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:fleetdm/confidential:*"]
    }
  }
}

resource "aws_iam_role" "fleetdm_confidential_cloudflare" {
  name               = "fleetdm-confidential-cloudflare-terraform"
  assume_role_policy = data.aws_iam_policy_document.fleetdm_confidential_cloudflare_trust.json
}

# --- Policy for backend S3/KMS/DynamoDB ---
data "aws_iam_policy_document" "fleetdm_confidential_cloudflare_rw" {
  # Full access to the specific Cloudflare tfstate object
  statement {
    sid = "S3ManageTfstate"
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:DeleteObject"
    ]
    resources = [
      "${module.remote-state-s3-backend.state_bucket.arn}/infrastructure/cloudflare/terraform.tfstate"
    ]
  }

  # Allow bucket listing and versioning checks
  statement {
    sid = "S3ListAndVersioning"
    actions = [
      "s3:ListBucket",
      "s3:GetBucketVersioning"
    ]
    resources = [module.remote-state-s3-backend.state_bucket.arn]
  }

  # KMS key usage for backend
  statement {
    sid = "KMSKeyAccess"
    actions = [
      "kms:Encrypt",
      "kms:Decrypt",
      "kms:DescribeKey",
      "kms:GenerateDataKey"
    ]
    resources = [module.remote-state-s3-backend.kms_key.arn]
  }

  # Allow listing KMS keys globally (safe)
  statement {
    sid = "KMSList"
    actions = [
      "kms:ListKeys"
    ]
    resources = ["*"]
  }

  # DynamoDB access for state locking
  statement {
    sid = "DynamoDBStateLock"
    actions = [
      "dynamodb:GetItem",
      "dynamodb:PutItem",
      "dynamodb:DeleteItem",
      "dynamodb:DescribeTable"
    ]
    resources = [module.remote-state-s3-backend.dynamodb_table.arn]
  }
}

resource "aws_iam_policy" "fleetdm_confidential_cloudflare_rw" {
  name   = "fleetdm-confidential-cloudflare-terraform"
  policy = data.aws_iam_policy_document.fleetdm_confidential_cloudflare_rw.json
}

resource "aws_iam_role_policy_attachment" "fleetdm_confidential_cloudflare_rw_attach" {
  role       = aws_iam_role.fleetdm_confidential_cloudflare.name
  policy_arn = aws_iam_policy.fleetdm_confidential_cloudflare_rw.arn
}


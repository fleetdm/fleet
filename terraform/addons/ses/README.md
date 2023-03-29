# ALB Logging Addon
This addon creates alb logging bucket(s) in s3 and optionally an athena database for those logs.

# Example Configuration

This assumes your fleet module is `main` and is configured with it's default documentation.

See https://github.com/fleetdm/fleet/blob/main/terraform/example/main.tf for details.

```
module "main" {
  source          = "github.com/fleetdm/fleet//terraform?ref=main"
  certificate_arn = module.acm.acm_certificate_arn
  vpc = {
    name = random_pet.main.id
  }
  fleet_config = {
    extra_environment_variables = module.firehose-logging.fleet_extra_environment_variables
    extra_iam_policies          = module.firehose-logging.fleet_extra_iam_policies
  }
  alb_config = {
    access_logs = {
      bucket  = module.logging_alb.log_s3_bucket_id
      prefix  = "fleet"
      enabled = true
    }
  }
}

module "logging_alb" {
  source        = "github.com/fleetdm/fleet//terraform/addons/logging-alb?ref=main"
  prefix        = "fleet"
  enable_athena = true
}
```

# Additional Information

Once this terraform is applied, the Athena table will need to be created.  See https://docs.aws.amazon.com/athena/latest/ug/application-load-balancer-logs.html for help with creating the table.

For this implementation, the S3 pattern for the `CREATE TABLE` query should look like this:

```
s3://your-alb-logs-bucket/<PREFIX>/AWSLogs/<ACCOUNT-ID>/elasticloadbalancing/<REGION>/
```

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.60.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_labels"></a> [labels](#module\_labels) | clouddrove/labels/aws | 1.3.0 |

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_route53_record.dkim](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route53_record) | resource |
| [aws_route53_record.ses_verification](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route53_record) | resource |
| [aws_route53_record.spf_domain](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route53_record) | resource |
| [aws_ses_domain_dkim.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ses_domain_dkim) | resource |
| [aws_ses_domain_identity.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ses_domain_identity) | resource |
| [aws_ses_domain_identity_verification.default](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ses_domain_identity_verification) | resource |
| [aws_iam_policy_document.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_route53_zone.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/route53_zone) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cname_type"></a> [cname\_type](#input\_cname\_type) | CNAME type for Record Set. | `string` | `"CNAME"` | no |
| <a name="input_domain"></a> [domain](#input\_domain) | Domain to use for SES. | `string` | n/a | yes |
| <a name="input_environment"></a> [environment](#input\_environment) | Environment (e.g. `prod`, `dev`, `staging`). | `string` | `""` | no |
| <a name="input_label_order"></a> [label\_order](#input\_label\_order) | Label order, e.g. `name`,`application`. | `list(any)` | `[]` | no |
| <a name="input_managedby"></a> [managedby](#input\_managedby) | ManagedBy, eg 'CloudDrove' | `string` | `"hello@clouddrove.com"` | no |
| <a name="input_name"></a> [name](#input\_name) | Name  (e.g. `app` or `cluster`). | `string` | `""` | no |
| <a name="input_repository"></a> [repository](#input\_repository) | Terraform current module repo | `string` | `"https://github.com/clouddrove/terraform-aws-ses"` | no |
| <a name="input_txt_type"></a> [txt\_type](#input\_txt\_type) | Txt type for Record Set. | `string` | `"TXT"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_fleet_extra_environment_variables"></a> [fleet\_extra\_environment\_variables](#output\_fleet\_extra\_environment\_variables) | n/a |
| <a name="output_fleet_extra_iam_policies"></a> [fleet\_extra\_iam\_policies](#output\_fleet\_extra\_iam\_policies) | n/a |

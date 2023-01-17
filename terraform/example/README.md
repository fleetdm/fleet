# Fleet Terraform Module Example
This code provides some example usage of the Fleet Terraform module, including how some addons can be used to extend functionality.

Due to Terraform issues, this code requires 3 applies "from scratch":
- terraform apply -target random\_pet.main
- terraform apply -target module.main.module.vpc

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~> 4.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.48.0 |
| <a name="provider_random"></a> [random](#provider\_random) | 3.4.3 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_acm"></a> [acm](#module\_acm) | terraform-aws-modules/acm/aws | 4.3.1 |
| <a name="module_firehose-logging"></a> [firehose-logging](#module\_firehose-logging) | ../addons/logging-destination-firehose | n/a |
| <a name="module_main"></a> [main](#module\_main) | ../ | n/a |

## Resources

| Name | Type |
|------|------|
| [aws_route53_record.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route53_record) | resource |
| [random_pet.main](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/pet) | resource |
| [aws_route53_zone.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/route53_zone) | data source |

## Inputs

No inputs.

## Outputs

No outputs.

# Fleet Terraform Module Example
This code provides some example usage of the Fleet Terraform module, including how some addons can be used to extend functionality.

Due to Terraform issues, this code requires 3 applies "from scratch":
1. `terraform apply -target module.fleet.module.vpc`
2. `terraform apply -target module.fleet`
3. `terraform apply`

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | 5.36.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 5.36.0 |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_acm"></a> [acm](#module\_acm) | terraform-aws-modules/acm/aws | 4.3.1 |
| <a name="module_fleet"></a> [fleet](#module\_fleet) | github.com/fleetdm/fleet//terraform | tf-mod-root-v1.7.1 |
| <a name="module_migrations"></a> [migrations](#module\_migrations) | github.com/fleetdm/fleet//terraform/addons/migrations | tf-mod-addon-migrations-v2.0.0 |

## Resources

| Name | Type |
|------|------|
| [aws_route53_record.main](https://registry.terraform.io/providers/hashicorp/aws/5.36.0/docs/resources/route53_record) | resource |
| [aws_route53_zone.main](https://registry.terraform.io/providers/hashicorp/aws/5.36.0/docs/resources/route53_zone) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_domain_name"></a> [domain\_name](#input\_domain\_name) | domain name to host fleet under | `string` | n/a | yes |
| <a name="input_vpc_name"></a> [vpc\_name](#input\_vpc\_name) | name of the vpc to provision | `string` | `"fleet"` | no |
| <a name="input_zone_name"></a> [zone\_name](#input\_zone\_name) | the name to give to your hosted zone | `string` | `"fleet"` | no |

## Outputs

No outputs.

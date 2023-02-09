# MDM addon
This addon enables MDM functionality for Fleet. It does this via several secrets in AWS that stores the necessary values.
The following secrets are created:
- dep
- scep
- apn

Since this module cannot determine the value for the secrets at apply time, this module must be applied in 2 phases:

1. In the first phase, just add the module without passing additional config to the main Fleet module
1. In the second phase, after the secret values have been populated, apply while also passing the additional config to the main Fleet module.

The secrets should have the following layouts, note that all values are strings. If a value is a JSON object, string escape it.:
## DEP
```
{
    "token": <token>,
    "cert": <cert>,
    "key": <key>,
    "token-encrypted": <key>
}
```

## SCEP
```
{
    "crt": <crt>,
    "key": <key>,
    "challenge": <challenge>
}
```

## APN
```
{
    "FLEET_MDM_APPLE_MDM_PUSH_CERT_PEM": <cert>,
    "FLEET_MDM_APPLE_MDM_PUSH_KEY_PEM": <privkey>
}
```

## Requirements

No requirements.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | n/a |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_secretsmanager_secret.apn](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret.dep](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret.scep](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_apn_secret_name"></a> [apn\_secret\_name](#input\_apn\_secret\_name) | n/a | `string` | `"fleet-apn"` | no |
| <a name="input_dep_secret_name"></a> [dep\_secret\_name](#input\_dep\_secret\_name) | n/a | `string` | `"fleet-dep"` | no |
| <a name="input_scep_secret_name"></a> [scep\_secret\_name](#input\_scep\_secret\_name) | n/a | `string` | `"fleet-scep"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_extra_environment_variables"></a> [extra\_environment\_variables](#output\_extra\_environment\_variables) | n/a |
| <a name="output_extra_secrets"></a> [extra\_secrets](#output\_extra\_secrets) | n/a |

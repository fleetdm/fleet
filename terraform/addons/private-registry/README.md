# Private Container Image Registry

This addon will provision the correct IAM policy to attach to the Fleet config for the ECS task definition
to utilize private registry credentials when pulling container images.

## Using a private container image repository

First create an AWS Secrets Manager Secret with your preferred method, for example:
```shell
aws secretsmanager create-secret --name MyRegistryCredentials \
    --description "Private registry credentials" \
    --secret-string '{"username":"<your_username>","password":"<your_password>"}'
```

Then provide this secret's ARN as the input to the variable `secret_arn`.

### Using in Fleet Config

```hcl
module "private-auth" {
  source     = "github.com/fleetdm/fleet//terraform/addons/private-registry"
  secret_arn = "arn:aws:secretsmanager:us-east-2:123456789:secret:MyRegistryCredentials"
}

module "main" {
  source       = "github.com/fleetdm/fleet//terraform"
  fleet_config = {
    # other fleet configs
    extra_execution_iam_policies = concat(module.private-auth.extra_execution_iam_policies /*, additional execution policies*/)
    repository_credentials       = "arn:aws:secretsmanager:us-east-2:123456789:secret:MyRegistryCredentials"
  }
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
| [aws_iam_policy.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_policy_document.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_secret_arn"></a> [secret\_arn](#input\_secret\_arn) | ARN of the AWS Secrets Manager secret that stores the private registry credentials | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_extra_execution_iam_policies"></a> [extra\_execution\_iam\_policies](#output\_extra\_execution\_iam\_policies) | n/a |
| <a name="output_secret_arn"></a> [secret\_arn](#output\_secret\_arn) | n/a |

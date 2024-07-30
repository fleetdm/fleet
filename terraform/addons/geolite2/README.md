# Geoip Terraform module for Fleet

This module adds Geoip data to the Fleet docker image for use with the Fleet Terraform module.

See the [documentation](https://fleetdm.com/docs/configuration/fleet-server-configuration#geoip) for some basic information about what happens under the hood.

You will need to supply a Maxmind license key and a destination docker registry (such as ECR) to hold the new image.

Outputs will be added to the environment variables in Fleet via the `extra_environment_variables` list.

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_docker"></a> [docker](#requirement\_docker) | 3.0.2 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_docker"></a> [docker](#provider\_docker) | 3.0.2 |
| <a name="provider_local"></a> [local](#provider\_local) | 2.4.1 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [docker_image.maxmind_fleet](https://registry.terraform.io/providers/kreuzwerker/docker/3.0.2/docs/resources/image) | resource |
| [docker_registry_image.maxmind_fleet](https://registry.terraform.io/providers/kreuzwerker/docker/3.0.2/docs/resources/registry_image) | resource |
| [local_file.dockerfile](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_destination_image"></a> [destination\_image](#input\_destination\_image) | n/a | `string` | n/a | yes |
| <a name="input_fleet_image"></a> [fleet\_image](#input\_fleet\_image) | n/a | `string` | n/a | yes |
| <a name="input_license_key"></a> [license\_key](#input\_license\_key) | n/a | `string` | n/a | yes |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_extra_environment_variables"></a> [extra\_environment\_variables](#output\_extra\_environment\_variables) | n/a |

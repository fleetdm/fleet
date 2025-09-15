# Deploy osquery perf to a Loadtest environment

# Before we begin

Although deployments through the github action should be prioritized, for manual deployments you will need.

- [A loadtest environment](../infra/README.md)
- Terraform v1.10.2
- Docker
- Go

# Deploy with Github Actions

> Note: All deployments should occur from main.

1. [Navigate to the github action](https://github.com/fleetdm/fleet/actions/workflows/loadtest-osquery-perf.yml)

2. On the top right corner, select the `Run Workflow` dropdown.

3. Fill out the details for the deployment.

4. After all details have been filled out, you will hit the green `Run Workflow` button, directly under the inputs. For `terraform_action` select `Plan`, `Apply`, or `Destroy`.
    - Plan will show you the results of a dry-run
    - Apply will deploy changes to the environment
    - Destroy will destroy your environment

# Deploy osquery perf manually

1. Clone the repository

2. Initialize terraform

    ```sh
    terraform init
    ```

3. Create a new the terraform workspace or select an existing workspace for your environment. The terraform workspace will be used in different area's of Terraform to drive uniqueness and access to the environment.

    > Note: The workspace from the infrastructure deployment will not be carried over to this deployment. A new or existing workspace, specifically for osquery perf must be used.
    >
    > Your workspace name must match the workspace name that was used for the infrastructure deployment. Failure to use a matching workspace name can lead to deployments in another environment.

    ```sh
    terraform workspace new <workspace_name>
    ```

    or, if your workspace already exists

    ```sh
    terraform workspace list
    terraform workspace select <workspace_name>
    ```

4. Ensure that your new or existing workspace is in use.

    ```sh
    terraform workspace show
    ```

5. Deploy the environment (will also trigger migrations automatically)

    > Note: Terraform will prompt you for confirmation to trigger the deployment. If everything looks ok, submitting `yes` will trigger the deployment.

    ```sh
    terraform apply -var=tag=v4.73.0 -var=git_branch=fleet-v4.73.0
    ```

    or, you can add the additional supported terraform variables, to overwrite the default values. You can choose which ones are included/overwritten. If a variable is not defined, the default value configured in [./variables.tf](variables.tf) is used.

    Below is an example with all available variables.

    ```sh
    terraform apply -var=tag=v4.73.0 -var=git_branch=fleet-v4.73.0 -var=loadtest_containers=20 -var=extra_flags=["--orbit_prob", "0.0"]
    ```

# Destroy osquery perf manually

1. Clone the repository (if not already cloned)

2. Initialize terraform

    ```sh
    terraform init
    ```

3. Select your workspace

    ```sh
    terraform workspace list
    terraform workspace select <workspace_name>
    ```

3. Destroy the environment

    ```sh
    terraform destroy
    ```

# Delete the workspace

Once all resources have been removed from the terraform workspace, remove the terraform workspace.

```sh
terraform workspace delete <workspace_name>
```

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | >= 5.68.0 |
| <a name="requirement_docker"></a> [docker](#requirement\_docker) | ~> 2.16.0 |
| <a name="requirement_git"></a> [git](#requirement\_git) | ~> 0.1.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 6.13.0 |
| <a name="provider_docker"></a> [docker](#provider\_docker) | 2.16.0 |
| <a name="provider_git"></a> [git](#provider\_git) | 0.1.0 |
| <a name="provider_terraform"></a> [terraform](#provider\_terraform) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_osquery_perf"></a> [osquery\_perf](#module\_osquery\_perf) | github.com/fleetdm/fleet-terraform//addons/osquery-perf | tf-mod-addon-osquery-perf-v1.1.1 |

## Resources

| Name | Type |
|------|------|
| [docker_registry_image.loadtest](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/resources/registry_image) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_ecr_authorization_token.token](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ecr_authorization_token) | data source |
| [aws_ecr_repository.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ecr_repository) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |
| [docker_registry_image.dockerhub](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/data-sources/registry_image) | data source |
| [git_repository.tf](https://registry.terraform.io/providers/paultyng/git/latest/docs/data-sources/repository) | data source |
| [terraform_remote_state.infra](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/data-sources/remote_state) | data source |
| [terraform_remote_state.shared](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/data-sources/remote_state) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_extra_flags"></a> [extra\_flags](#input\_extra\_flags) | Comma delimited list (string) for passing extra flags to osquery-perf containers | `list(string)` | <pre>[<br/>  "--orbit_prob",<br/>  "0.0"<br/>]</pre> | no |
| <a name="input_git_branch"></a> [git\_branch](#input\_git\_branch) | The git branch to use to build loadtest containers.  Only needed if docker tag doesn't match the git branch | `string` | `null` | no |
| <a name="input_loadtest_containers"></a> [loadtest\_containers](#input\_loadtest\_containers) | Number of loadtest containers to deploy | `number` | `1` | no |
| <a name="input_tag"></a> [tag](#input\_tag) | The tag to deploy. This would be the same as the branch name | `string` | `""` | no |

## Outputs

No outputs.

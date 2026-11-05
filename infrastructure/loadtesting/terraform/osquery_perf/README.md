# Deploy osquery perf to a Loadtest environment

# Before we begin

Although deployments through the github action should be prioritized, for manual deployments you will need.

- [A loadtest environment](../infra/README.md)
- Terraform v1.10.2
- Docker
- Go

# Deploy with Github Actions

1. [Navigate to the github action](https://github.com/fleetdm/fleet/actions/workflows/loadtest-osquery-perf.yml)

2. On the top right corner, select the `Run Workflow` dropdown.

3. Fill out the details for the deployment.
    > Note: For mdm testing, where a SCEP CHALLENGE secret is required, you will need to fetch the details from the database [using this script](https://github.com/fleetdm/fleet/blob/main/tools/mdm/assets/main.go) and passing it in as an `extra_flag` when triggering an osquery\_perf deployment.
    >
    > Requires:
    > - VPN connection
    > - Database access

   Example:
   ```
   ["--mdm_scep_challenge", "<value>"]
   ```

   Additional parameters can be passed in (and not limited to):

   ```
   ["--logger_tls_period", "120s", "--orbit_prob", "0.0", "--http_message_signature_prob", "0.0", "--mdm_prob", "1.0", "--os_templates", "<value>", "--mdm_scep_challenge", "<value>"]
   ```

> **IMPORTANT**: osquery\_perf needs to be destroyed before the infrastructure

4. After all details have been filled out, you will hit the green `Run Workflow` button, directly under the inputs. For `terraform_action` select `Plan`, `Apply`, or `Destroy`.
    - `Plan` will show you the results of a dry-run
    - `Apply` will deploy changes to the environment
    - `Destroy` will destroy your environment

# Stable hosts and realistic host behavior

These options make simulated hosts behave more like a real fleet. All of them are off by default, so existing load tests behave as before. The flags need an osquery-perf build that has them: osquery-perf exits on flags it doesn't know.

**Stable host identities.** Set the workflow's `stable_host_identities` input (or `-var=stable_host_identities=true`) to run each container as its own ECS service with its own `--seed`. A container that ECS replaces, or that you redeploy with a new osquery-perf build or new flags, comes back as the same hosts instead of enrolling new ones. Pass `--http_message_signature_prob 0` with it: hosts that sign their requests can't re-enroll after a restart. MDM hosts re-enroll after a restart, so Fleet resends their profiles.

**Host behavior flags.** Add these to `extra_flags`. The suggested values are deliberately worse than a typical fleet, but each one still happens in real fleets.

| Flag | Default | Suggested | What it does |
|---|---|---|---|
| `--policy_sticky` | off | on | Each host decides once per policy (passing with `--policy_pass_prob`) instead of re-rolling on every report. |
| `--software_churn_per_day` | off (`-1`) | `0.05` | Keeps each host's software inventory stable and replaces this share of it a day, across restarts too. `0.05` is a developer-heavy fleet: weekly Homebrew upgrades and auto-updating apps and browser extensions. Off, about a third of Windows and Ubuntu reports replace the whole inventory. |
| `--software_query_fail_prob` | `0.5` | `0.05` | Share of software reports that fail. Fleet skips failed reports, so the default halves software ingestion. |
| `--last_opened_change_prob` | `0.5` | `0.05` | Chance that each of a Mac's 50 tracked apps was opened since its last report. `0.05` is about 55 apps a day, like a shared lab Mac. |
| `--software_installer_permafail_prob` | `0` | `0.10` | Share of hosts whose software installs always fail, like a package that can't install on part of the fleet. Exercises Fleet's install retry limit. |
| `--script_exec_fail_prob` | `0.5` | default | Share of script runs that exit with code 1. |

Example `extra_flags`:

```
["--orbit_prob", "0.0", "--http_message_signature_prob", "0", "--policy_sticky", "--software_churn_per_day", "0.05", "--software_query_fail_prob", "0.05", "--last_opened_change_prob", "0.05", "--software_installer_permafail_prob", "0.10"]
```

MDM Macs also report declaration (DDM) status and escrow FileVault keys that Fleet can decrypt. These don't need a flag.

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
    terraform apply -var=git_tag_branch=fleet-v4.76.0
    ```

    or, you can add the additional supported terraform variables, to overwrite the default values. You can choose which ones are included/overwritten. If a variable is not defined, the default value configured in [./variables.tf](variables.tf) is used.

    > Note: For mdm testing, where a SCEP CHALLENGE secret is required, you will need to fetch the details from the database [using this script](https://github.com/fleetdm/fleet/blob/main/tools/mdm/assets/main.go) and passing it in as an `extra_flag` when triggering an osquery\_perf deployment.
    >
    > Requires:
    > - VPN connection
    > - Database access

    Example with the SCEP CHALLENGE secret as an extra\_flag:
    
    ```sh
    terraform apply -var=git_tag_branch=fleet-v4.76.0 -var=extra_flags=["--mdm_scep_challenge", "<value>"]
    ```

    Example with all available variables.

    ```sh
    terraform apply -var=git_tag_branch=fleet-v4.76.0 -var=loadtest_containers=20 -var=extra_flags=["--orbit_prob", "0.0"]
    ```

6. If you'd like to deploy osquery\_perf tasks in batches, you can now run the original `enroll.sh` script, from the osquery\_perf directory. The script will deploy in batches of 8, every 60 seconds, so it's recommended to set your starting index and max number of osquery perf containers as a multiple of 8.

   ```sh
   ./enroll.sh <branch_or_tag_name> <starting index> <max number of osquery_perf containers> <sleep_time>
   ```

# Destroy osquery perf manually

> **IMPORTANT**: osquery\_perf needs to be destroyed before the infrastructure

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
| <a name="requirement_docker"></a> [docker](#requirement\_docker) | ~> 3.6.0 |
| <a name="requirement_git"></a> [git](#requirement\_git) | 2025.10.10 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 6.23.0 |
| <a name="provider_docker"></a> [docker](#provider\_docker) | 3.6.2 |
| <a name="provider_git"></a> [git](#provider\_git) | 2025.10.10 |
| <a name="provider_random"></a> [random](#provider\_random) | 3.7.2 |
| <a name="provider_terraform"></a> [terraform](#provider\_terraform) | n/a |

## Modules

| Name | Source | Version |
|------|--------|---------|
| <a name="module_osquery_perf"></a> [osquery\_perf](#module\_osquery\_perf) | github.com/fleetdm/fleet-terraform//addons/osquery-perf | tf-mod-addon-osquery-perf-v1.2.1 |

## Resources

| Name | Type |
|------|------|
| [aws_ecr_repository.loadtest](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecr_repository) | resource |
| [aws_ecs_service.osquery_perf_stable](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_service) | resource |
| [aws_ecs_task_definition.osquery_perf_stable](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/ecs_task_definition) | resource |
| [aws_kms_key.main](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/kms_key) | resource |
| [docker_image.loadtest](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/resources/image) | resource |
| [docker_registry_image.loadtest](https://registry.terraform.io/providers/kreuzwerker/docker/latest/docs/resources/registry_image) | resource |
| [random_pet.rand_image_key](https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/pet) | resource |
| [aws_caller_identity.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/caller_identity) | data source |
| [aws_ecr_authorization_token.token](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ecr_authorization_token) | data source |
| [aws_ecr_repository.fleet](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/ecr_repository) | data source |
| [aws_region.current](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/region) | data source |
| [aws_secretsmanager_secret_version.enroll_secret](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/secretsmanager_secret_version) | data source |
| [git_repository.tf](https://registry.terraform.io/providers/metio/git/2025.10.10/docs/data-sources/repository) | data source |
| [terraform_remote_state.infra](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/data-sources/remote_state) | data source |
| [terraform_remote_state.shared](https://registry.terraform.io/providers/hashicorp/terraform/latest/docs/data-sources/remote_state) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_extra_flags"></a> [extra\_flags](#input\_extra\_flags) | Comma delimited list (string) for passing extra flags to osquery-perf containers | `list(string)` | <pre>[<br/>  "--orbit_prob",<br/>  "0.0"<br/>]</pre> | no |
| <a name="input_git_tag_branch"></a> [git\_tag\_branch](#input\_git\_tag\_branch) | The tag or git branch to use to build loadtest containers. | `string` | n/a | yes |
| <a name="input_loadtest_containers"></a> [loadtest\_containers](#input\_loadtest\_containers) | Number of loadtest containers to deploy | `number` | `1` | no |
| <a name="input_stable_host_identities"></a> [stable\_host\_identities](#input\_stable\_host\_identities) | Run each container as its own service with its own --seed, so a restarted or replaced container comes back as the same hosts instead of enrolling new ones | `bool` | `false` | no |
| <a name="input_task_size"></a> [task\_size](#input\_task\_size) | n/a | <pre>object({<br/>    cpu    = optional(number, 256)<br/>    memory = optional(number, 1024)<br/>  })</pre> | <pre>{<br/>  "cpu": 256,<br/>  "memory": 1024<br/>}</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_osquery_perf"></a> [osquery\_perf](#output\_osquery\_perf) | n/a |

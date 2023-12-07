# Releasing Fleet

## Release process

This section outlines the release process at Fleet after all EMs have certified that they are ready for release. 

The current release cadence is once every three weeks. Patch versions are released as needed.

### Prepare a new version of Fleet

Note: Please prefix versions with `fleet-v` (e.g., `fleet-v4.0.0`) in git tags, Helm charts, and NPM configs.

1. Update the [CHANGELOG](https://github.com/fleetdm/fleet/blob/main/CHANGELOG.md) with the changes you made since the last
   Fleet release. Use `make changelog` to pull the changes files into `CHANGELOG.md`, then manually
   edit. When editing, order the most relevant/important changes at the time and try to make the
   tone and syntax of the written language match throughout the document. `make changelog` will stage all changes
   file entries for deletion with the commit.

2. Update version numbers in the relevant files:

   - [fleetctl package.json](https://github.com/fleetdm/fleet/blob/main/tools/fleetctl-npm/package.json) (do not yet `npm publish`)
   - [Helm chart.yaml](https://github.com/fleetdm/fleet/blob/main/charts/fleet/Chart.yaml) and [values file](https://github.com/fleetdm/fleet/blob/main/charts/fleet/values.yaml)
   - Terraform variables ([AWS](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/variables.tf)/[GCP](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/gcp/variables.tf))
   - [Kubernetes `deployment.yml` example file](https://github.com/fleetdm/fleet/blob/main/docs/Deploy/Deploying-Fleet-on-Kubernetes.md)

   Commit these changes via Pull Request and pull the changes on the `main` branch locally. Check that
   `HEAD` of the `main` branch points to the commit with these changes.

### Prepare a minor or major release

1. Complete the steps above to [prepare a new version of Fleet](#prepare-a-new-version-of-fleet).

2. Create a a new branch. Minor or major release branches should be prefixed with `prepare-`. In this example we are creating `4.3.1`:
   ```sh
   git checkout main
   git checkout --branch prepare-fleet-v4.3.1
   ```

3. Tag and push the new release in Git:
   ```sh
   git tag fleet-v<VERSION>
   git push origin fleet-v<VERSION>
   ```

   Note that `origin` may be `upstream` depending on your `git remote` configuration. The intent here
   is to push the new tag to the `github.com/fleetdm/fleet` repository.

   After the tag is pushed, GitHub Actions will automatically begin building the new release.

   ***

   Wait while GitHub Actions creates and uploads the artifacts.

   ***

   When the Actions Workflow has been completed, publish the new version of Fleet.

### Preparing a patch release

A patch release is required when a critical bug is found. Critical bugs are defined in [our handbook](https://fleetdm.com/handbook/quality#critical-bugs).

1. Complete the steps above to [prepare a new version of Fleet](#prepare-a-new-version-of-fleet).

2. Create a new branch, starting from the git tag of the prior release. Patch branches should be prefixed with `patch-`. In this example we are creating `4.3.1`:
   ```sh
   git checkout fleet-v4.3.0
   git checkout --branch patch-fleet-v4.3.1
   ```

3. Cherry picks the necessary commits from `main` into the new branch:
   ```sh
   git cherry-pick d34db33f
   ```

> Make sure to cherry-pick the commit containing changelog and version number updates.

4. Push the branch to [fleetdm/fleet](https://github.com/fleetdm/fleet).
   ```sh
   git push origin patch-fleet-v4.3.1
   ```

   When a `patch-*` branch is pushed, the [Docker publish
   Action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) will
   run and create a container image for QA with `fleetctl preview` (eg. `fleetctl preview --tag patch-fleet-v4.3.1`).

5. Check the [Docker Publsih GitHub action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) to confirm it completes successfully for this branch.

5. Create a [Release QA](https://github.com/fleetdm/fleet/blob/main/.github/ISSUE_TEMPLATE/smoke-tests.md) issue. Populate the version and browsers, and assign to the QA person leading the release. Add the appropriate [product group label](https://fleetdm.com/handbook/company/product-groups), and `:release` label, so that it appears on the product group's release board.

6. QA conducts release tests. When they all pass, the patch is ready for release. 

7. **Important!** The DRI for creating the patch release branch manually checks the database migrations. Any migrations that are not cherry-picked in a patch must have a _later_ timestamp than migrations that were cherry-picked. If there are new migrations that were not cherry-picked, verify that those migrations have later timestamps. If they do not, submit a new Pull Request to increase the timestamps and ensure that migrations are run in the appropriate order.

8. Tag and push the new release in Git:
   ```sh
   git tag fleet-v-v4.3.1
   git push origin fleet-v-4.3.1
   ```

   Note that `origin` may be `upstream` depending on your `git remote` configuration. The intent here
   is to push the new tag to the `github.com/fleetdm/fleet` repository.

   After the tag is pushed, GitHub Actions will automatically begin building the new release.

   ***

   Wait while GitHub Actions creates and uploads the artifacts.

   ***

   When the Actions Workflow has been completed, [publish the new version of Fleet](#publish-a-new-version-of-fleet).

### Publish a new version of Fleet

4. Edit the draft release on the [GitHub releases page](https://github.com/fleetdm/fleet/releases).
   Use the version number as the release title. Use the below template for the release description
   (replace items in <> with the appropriate values):
   ```md
   ### Changes

   <COPY FROM CHANGELOG>

   ### Upgrading

   Please visit our [update guide](https://fleetdm.com/docs/deploying/upgrading-fleet) for upgrade instructions.

   ### Documentation

   Documentation for Fleet is available at [fleetdm.com/docs](https://fleetdm.com/docs).

   ### Binary Checksum

   **SHA256**

   <COPY FROM checksums.txt>
   ```

   When editing is complete, publish the release.

5. Publish the new version of `fleetctl` on NPM. Run `npm publish` in the
   [fleetctl-npm](https://github.com/fleetdm/fleet/tree/main/tools/fleetctl-npm) directory. Note that NPM does not allow replacing a
   package without creating a new version number. Take care to get things correct before running
   `npm publish`!

> If releasing a "prerelease" of Fleet, run `npm publish --tag prerelease`. This way, you can
> publish a prerelease of fleetctl while the most recent fleetctl npm package, available for public
> download, is still the latest _official_ release.

6. Deploy the new version to Fleet's internal dogfood instance: https://fleetdm.com/handbook/engineering#deploying-to-dogfood.

7. In the #g-infra Slack channel, notify the @infrastructure-oncall of the release. This way, the @infrastructure-oncall individual can deploy the new version.

8. Announce the release in the #general channel. 

9. Announce the release in the #fleet channel of [osquery Slack](https://fleetdm.com/slack) by updating the channel topic with the link to this release. 

<meta name="pageOrderInSection" value="500">
<meta name="description" value="Learn how new versions of Fleet are tested and released.">

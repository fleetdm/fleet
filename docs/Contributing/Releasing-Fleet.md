# Releasing Fleet

## Release process

This section outlines the release process at Fleet after all EMs have certified that they are ready for release. 

The current release cadence is once every three weeks. Patch versions are released as needed.
#### Release candidate process

Major, minor, and patch releases go through the same release candidate and smoke testing process prior to release. 

1. The release ritual or patch release DRI creates a release candidate branch. If this is a major or minor version, they prepend the branch name with `prepare-v`. If it is a patch version, they prepend the branch name with `patch-v`. For example, `prepare-v4.0.0` or `patch-v4.0.1`.

> When a `prepare-*` or `patch-*` branch is pushed, the [Docker publish Action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) will be invoked to push a container image for smoke testing with `fleetctl preview` (eg. `fleetctl preview --tag patch-fleet-v4.3.1`).

2. The release DRI pushes the branch to github.com/fleetdm/fleet:
   ```
   git push origin prepare-fleet-v4.0.0
   ```

3. The release DRI checks in the GitHub UI that Actions ran successfully for this branch.

4. The release DRI adds a comment to the related smoke testing issue [created during the freeze ritual](https://fleetdm.com/handbook/engineering#create-release-qa-issue) confirming that the release candidate is ready for testing. 

5. QA is conducted on the release candidate following the steps in the smoke testing issue. If smoke testing passes successfully, the EM for each relevant product group adds a comment to the issue certifying that their product group has completed smoke testing. 

6. When the smoke testing issue has been certified, the release DRI publishes the release. 



### Preparing a minor or major release

Note: Please prefix versions with `fleet-v` (e.g., `fleet-v4.0.0`) in git tags, Helm charts, and NPM configs.

1. Update the [CHANGELOG](https://github.com/fleetdm/fleet/blob/main/CHANGELOG.md) with the changes you made since the last
   Fleet release. Use `make changelog` to pull the changes files into `CHANGELOG.md`, then manually
   edit. When editing, order the most relevant/important changes at the time and try to make the
   tone and syntax of the written language match throughout the document. `make changelog` will stage all changes
   file entries for deletion with the commit.

   Add a "Performance" section below the list of changes. This section should summarize the number of
   hosts that the Fleet server can handle, call out if this number has
   changed since the last release, and list the infrastructure used in the load testing environment.

   Update version numbers in the relevant files:

   - [fleetctl package.json](https://github.com/fleetdm/fleet/blob/main/tools/fleetctl-npm/package.json) (do not yet `npm publish`)
   - [Helm chart.yaml](https://github.com/fleetdm/fleet/blob/main/charts/fleet/Chart.yaml) and [values file](https://github.com/fleetdm/fleet/blob/main/charts/fleet/values.yaml)
   - Terraform variables ([AWS](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/variables.tf)/[GCP](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/gcp/variables.tf))
   - [Kubernetes `deployment.yml` example file](https://github.com/fleetdm/fleet/blob/main/docs/Deploy/Deploying-Fleet-on-Kubernetes.md)

   Commit these changes via Pull Request and pull the changes on the `main` branch locally. Check that
   `HEAD` of the `main` branch points to the commit with these changes.

2. Create release candidate branch and conduct smoke testing as documented above. 

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

   When the Actions Workflow has been completed:

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

9. Announce the release in the #fleet channel of [osquery
   Slack](https://fleetdm.com/slack) and
   update the channel's topic with the link to this release. Using `@here` requires admin
   permissions, so typically this announcement will be done by `@zwass`.

   Announce the release via blog post (on Medium) and Twitter (linking to blog post).

### Preparing a patch release

A patch release is required when a critical bug is found. Critical bugs are defined in [our handbook](https://fleetdm.com/handbook/quality#critical-bugs).

#### Process

1. The DRI for release testing/QA notifies the [directly responsible individual (DRI) for creating the patch release branch](https://fleetdm.com/handbook/engineering#rituals) to create the new branch, starting from the git tag of the prior release. Patch branches should be prefixed with `patch-`. In this example we are creating `4.3.1`:
   ```sh
   git checkout fleet-v4.3.0
   git checkout --branch patch-fleet-v4.3.1
   ```

2. The DRI for creating the patch release branch cherry picks the necessary commits into the new branch:
   ```sh
   git cherry-pick d34db33f
   ```

3. The DRI for creating the patch release branch pushes the branch to github.com/fleetdm/fleet:
   ```sh
   git push origin patch-fleet-v4.3.1
   ```

   When a `patch-*` branch is pushed, the [Docker publish
   Action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) will
   be invoked to push a container image for QA with `fleetctl preview` (eg. `fleetctl preview --tag patch-fleet-v4.3.1`).

4. The patch release DRI checks in the GitHub UI that Actions ran successfully for this branch.

5. The patch release DRI notifies the [DRI for release testing/QA](https://fleetdm.com/handbook/product#rituals) that the branch is available for completing [smoke tests](https://github.com/fleetdm/fleet/blob/main/.github/ISSUE_TEMPLATE/smoke-tests.md).

6. The DRI for release testing/QA makes sure the standard release instructions at the top of this document are followed. Be sure that modifications to the changelog and config files are commited _on the `patch-*` branch_.

7. The DRI for release testing/QA notifies the [DRI for the release ritual](https://fleetdm.com/handbook/engineering#rituals) that the patch release is ready. The DRI for the release ritual releases the patch.

8. The DRI for creating the patch release branch cherry-picks the commit containing the changelog updates into a new branch, and merges that commit into `main` through a Pull Request.

9. **Important!** The DRI for creating the patch release branch manually checks the database migrations. Any migrations that are not cherry-picked in a patch must have a _later_ timestamp than migrations that were cherry-picked. If there are new migrations that were not cherry-picked, verify that those migrations have later timestamps. If they do not, submit a new Pull Request to increase the timestamps and ensure that migrations are run in the appropriate order.

<meta name="pageOrderInSection" value="500">
<meta name="description" value="Learn how new versions of Fleet are tested and released.">

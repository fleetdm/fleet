Creating a new release
==================

Releases must be built locally(for now) because packages have to be signed and we're not in a position to trust a public CI system with signing keys.
Once packages are built and dep/apt repos are generated, the repo is synced to a GCS cloud bucket which makes the release immediately available at `dl.kolide.com`

# Requirements

A local copy of the google storage bucket. `gs://dl.kolide.co/`
GPG keys in your keyring.

# Steps

1. Download the Google Storage bucket locally.

```
gsutil cp -r gs://dl.kolide.co/ /Users/$user/kolide_packages/
```

2. Import keys to GPG keyring. Run this command by mounting the `~/.gnupg` folder into the `kolide/fpm` docker container. The gnupg version on your mac is probably different and the keyring format is not compatible with the one in the container. The permissions on .gnupg should be 700 and the files in the .gnupg directory need to be 600.

Note: You only need to do this step once.

Start container

```
	docker run --rm -it \
        -v /Users/$(whoami)/.gnupg:/root/.gnupg" \
        kolide/fpm /bin/bash
```

And in the container, run:

```
gpg --allow-secret-key-import --import private.key
```

Then check the key is there, with `gpg --list-keys`
You should see the Kolide packaging key there.


3. Build binaries/packages.
Use the `build_release.sh` script in this folder to create a zip of the binaries and linux packages. The scripts for building linux packages will run in a docker container, so if you're running for the first time, you might see the containers downloading.

You will be prompted for the GPG password several times by the rpm/deb packaging scripts.

4. Copy the artifacts into the appropriate directories in `~/kolide_packages`

Example:

```
cp build/kolide-1.0.4-1.x86_64.rpm ~/kolide_packages/yum/
cp build/kolide_1.0.4_amd64.deb ~/kolide_packages/deb
cp build/kolide_1.0.4.zip ~/kolide_packages/bin
cp build/kolide_latest.zip ~/kolide_packages/bin/kolide_latest.zip
```

5. Run the `update-package-repos` script. The script will update/sign the metadata for the local yum/apt repos. You will be prompted for the GPG key password again during this step so have it ready.
The script assumes the packages are in `LOCAL_REPO_PATH="/Users/${USER}/kolide_packages"`

NOTE: The script MUST be run from the `pkgrepos` directory as it `cd`s into relative folders. Should probably be fixed...

6. Generate the package metadate file.
The `https://dl.kolide.co/metadata.json` file holds data about the latest version/old releases. Run the Go `package_metadata.go` to generate an updated version of the metadata file.

```
go run package_metadata.go -repo /Users/$me/kolide_packages/ -git-tag=1.0.4
 ```

7. Create a git commit commit with the updated package repos.
The repo building scripts can be flaky, and occasionaly it's useful to use a `--reset HARD` flag with git to retry building the release.

8. Push the release to gcloud. Pushing will override the contents of the gcs bucket and the release will be immediately available.

```
gsutil -m rsync -r -d ./kolide_packages gs://dl.kolide.co/
```

# Testing

The `~/kolide_packages` folder has a nginx dockerfile which builds a static site of the repo. You can use it to host a local version of the yum/apt repo.


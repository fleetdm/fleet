# Fleetd updates 

The fleetd agent will periodically check the public Fleet update repository and update Orbit, Fleet Desktop, and/or osquery if it detects a later version.

To override this behavior, users can set a channel for each component or disable updates altogether. Visit [Adding Hosts](https://fleetdm.com/docs/using-fleet/adding-hosts#fleet-desktop) to learn more.
Alternatively, users with a Fleet Premium subscription can self-manage an update server.

## Securing updates

Fleetd utilizes [The Update Framework](https://theupdateframework.io/) to secure the update system. The TUF specification provides a robust framework for establishing trust over the content of updates. See [TUF's security documentation](https://theupdateframework.io/security/) for more details.

Fleet's usage of TUF allows the keys most critical to the security of the system to be stored offline, and provides a simple deployment model for update metadata and content.

There is no server that must be maintained for updates, instead Fleet provides tools via `fleetctl` to manage the static metadata and update assets. These can be served by any static content hosting solution (Apache, nginx, S3, etc.).

## Operations

Update management is handled by the `fleetctl updates` subcommands.

Fleet will prompt for passphrases when needed, or passphrases may be set in the environment variables `FLEET_ROOT_PASSPHRASE`, `FLEET_TARGETS_PASSPHRASE`, `FLEET_SNAPSHOT_PASSPHRASE`, and `FLEET_TIMESTAMP_PASSPHRASE`. Passphrases should be stored separately from keys.

By default, the current working directory is used for the TUF repository. All update commands support a `--path` parameter to use a different directory.

### Initialize the repository

_The root cryptographic key generated in this step is highly sensitive, and critical to the security of the update system. We recommend following these steps from a trusted, offline, ephemeral environment such as [Debian Live](https://www.debian.org/CD/live/) running from a USB stick. Avoid placing the root key in an online environment. Fleet will soon support the use of Hardware security modules (HSMs) to further protect the root key._

For testing purposes it is okay to initialize the repository in an online environment. Be sure to use a clean offline environment with new keys and passphrases when deploying to production.

Initialize the repository:

```sh
fleetctl updates init
```

Choose and record secure passphrases, _different for each key_. If the passphrases are not already set in the environment, you will be prompted to input them.

Make multiple copies of the `keys` directory to be stored offline on USB drives. These copies contain the root key:

```sh
cp -r keys <destination>
```

Delete the root key from the `keys` directory:

```sh
rm keys/root.json
```

Copy the `keys`, `repository`, and `staged` directories to a separate "working" USB drive:

```sh
cp -r keys repository staged <destination>
```

Shut down the environment.

### Deploy updates

Updates are deployed first by staging the contents and metadata, then publishing.

#### Staging
 
_Staging targets requires access to the `target`, `snapshot`, and `timestamp` keys. Best practice is to connect the drive containing the keys while staging updates and leave the keys offline at other times._

Use `fleetctl updates add` to stage updates. Fleetd updates the `osqueryd` binary, as well as the `orbit` binary. Updates are staged for each of these separately using the `--name` flag. It is not necessary to update both at the same time.

The following commands will prompt for key passphrases if not specified in the environment.

To stage updates for `osqueryd`:

```sh
fleetctl updates add --target ./path/to/linux/osqueryd  --platform linux --name osqueryd --version 4.6.0 -t 4.6 -t 4 -t stable 
```

This will add the `osqueryd` binary located at `./path/to/osqueryd` to the channels `4.6.0`, `4.6`, `4`, and `stable` for the `linux` platform.

In a typical scenario, each platform is staged before the repository is published.

Stage the equivalent macOS update:

```sh
fleetctl updates add --target ./path/to/macos/osqueryd  --platform macos --name osqueryd --version 4.6.0 -t 4.6 -t 4 -t stable 
```

A similar process can be used to stage the `orbit` artifacts by substituting `--name orbit`

When updates are staged, publish the repository.

#### Publishing

Publishing updates is as simple as making the contents of the `repository` directory available over HTTP. This can be achieved with [AWS S3](https://docs.aws.amazon.com/AmazonS3/latest/userguide/HostingWebsiteOnS3Setup.html), [Apache](https://access.redhat.com/solutions/67298), [NGINX](https://docs.nginx.com/nginx/admin-guide/web-server/serving-static-content/), or any other static file hosting solution or CDN.

Python's `SimpleHTTPServer` can be used for quick local testing:

```sh
cd repository && python -m SimpleHTTPServer
```

Or, for Python version 3.0 and greater:

```sh
cd repository && python -m http.server
```

Run this to host the repository at http://localhost:8000.

#### Update timestamp

Fleetd verifies freshness of the update metadata using the signed [timestamp file](https://theupdateframework.io/metadata/#timestamp-metadata-timestampjson). _This file must be re-signed every two weeks_ (this interval will be made configurable soon).

To update the timestamp metadata:

```sh
fleetctl updates timestamp
```

_This operation requires the `timestamp` key to be available, along with the corresponding passphrase. Best practice is to keep these keys "online" in a context where they can be used to update the metadata on an interval (via `cron`, AWS Lambda, etc.). This "online" context should be on a separate host from the static file server, to prevent leaking these less sensitive (though still sensitive) keys in the event the static file server is compromised._

### Building packages

Note that `osqueryd` and `orbit` updates must be published before packages can be produced.

Record the root key metadata with a copy of the repository:

```sh
fleetctl updates roots
```

This output is _not sensitive_ and will be shared in agent deployments to verify the contents of updates and metadata. Provide the JSON output in the `--update-roots` flag of the fleetd agent.

### Packaging with fleetd

You can use `fleetctl package` to generate Fleet's agent (fleetd) to integrate with your Fleet instance.

For example running `fleetctl package --type deb --fleet-url=<fleet url> --enroll-secret=<enroll secret>` will build a `.deb` installer with everything needed
to communicate with your fleet instance.

See the [Enroll hosts docs](https://fleetdm.com/docs/using-fleet/enroll-hosts) for instructions on generating the fleetd agent.

### Key rotation

Key rotation is supported for each of the update role keys via the `fleetctl updates rotate` command.

Rotation is required for a key if the key has been compromised, or before the key expires.

Compromise of a single key (besides the root key) within the system does not enable an attacker to
push arbitrary updates. Compromise of the root key is a catastrophic failure allowing arbitrary
updates, and for this reason the root key is highly guarded in an offline context. See Section 7.4
of the [_Survivable Key
Compromise_](https://theupdateframework.io/papers/survivable-key-compromise-ccs2010.pdf) paper for a
more in-depth discussion of the implications of key compromise in the TUF system.

To rotate (for example) the targets key:

```sh
fleetctl updates rotate targets
```

After the key(s) have been rotated, publish the repository in the same fashion as any other update.

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="authorFullName" value="Noah Talerman">
<meta name="publishedOn" value="2024-04-30">
<meta name="articleTitle" value="Fleetd updates">
<meta name="description" value="Information on how to manage and secure Fleet agent updates.">

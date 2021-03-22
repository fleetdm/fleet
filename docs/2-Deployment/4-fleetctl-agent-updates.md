# Self-managed agent updates 

Fleet's [Orbit](https://github.com/fleetdm/fleet) osquery updater by default utilizes the public Fleet update repository.

For users who would like to self-manage an update server, this capability is available with the Fleet Core subscription.

## Securing updates

Orbit utilizes [The Update Framework](https://theupdateframework.io/) to secure the update system. The TUF specification provides a robust framework for establishing trust over the content of updates. See [TUF's security documentation](https://theupdateframework.io/security/) for more details.

Fleet's usage of TUF allows the keys most critical to the security of the system to be stored offline, and provides a simple deployment model for update metadata and content.

There is no server that must be maintained for updates, instead Fleet provides tools via `fleetctl` to manage the static metadata and update assets. These can be served by any static content hosting solution (Apache, nginx, S3, etc.).

## Operations

Update management is handled by the `fleetctl updates` subcommands.

Fleet will prompt for passphrases when needed, or passphrases may be set in the environment variables `FLEET_ROOT_PASSPHRASE`, `FLEET_TARGETS_PASSPHRASE`, `FLEET_SNAPSHOT_PASSPHRASE`, and `FLEET_TIMESTAMP_PASSPHRASE`. Passphrases should be stored separately from keys.

By default, the current working directory is used for the TUF repository. All update commands support a `--path` parameter to use a different directory.

### Initialize the repository

_Note: The root cryptographic key generated in this step is highly sensitive, and critical to the security of the update system. We recommend following these steps from a trusted, offline, ephemeral environment such as [Debian Live](https://www.debian.org/CD/live/) running from a USB stick. Avoid placing the root key in an online environment. Fleet will soon support the use of Hardware security modules (HSMs) to further protect the root key._

Initialize the repository:

```
fleetctl updates init
```

Choose and record secure passphrases, _different for each key_. If the passphrases are not already set in the environment, you will be prompted to input them.

Make multiple copies of the `keys` directory to be stored offline on USB drives. These copies contain the root key:

```
cp -r keys <destination>
```

Delete the root key from the `keys` directory:

```
rm keys/root.json
```

Copy the `keys`, `repository`, and `staged` directories to a separate "working" USB drive:

```
cp -r keys repository staged <destination>
```

Shut down the environment.

### 

Record the root key metadata:

```
fleetctl updates roots
```

This output is not sensitive and will be shared in all client deployments.

# Enroll Linux hosts on first boot with cloud-init

Fleet has no built-in zero-touch enrollment for Linux. This guide shows how to build it with cloud-init, the first-boot tool that ships in most Linux cloud images and in Ubuntu's unattended installer. A new host installs fleetd on its first boot and appears in Fleet about 20 to 30 seconds later, with nobody touching it.

cloud-init isn't part of Fleet, so you run the supporting infrastructure yourself. This guide was tested with Ubuntu 24.04 and AlmaLinux 10 on arm64, Fleet 4.91.1, and fleetd 1.61.0.

## What you provide

- An image that runs cloud-init on first boot. Public cloud images and other VM images do. Installs from a desktop ISO generally don't, so for physical machines use [Ubuntu autoinstall](#physical-machines-with-ubuntu-autoinstall), where the installer hands your user-data to cloud-init for one first boot and then disables it. To check an image, run `cloud-init status` on it.
- A way to hand user-data to the host: your cloud provider's user-data field, a seed volume labeled `CIDATA`, or autoinstall.
- A place to host the fleetd package that new hosts can reach over HTTPS at first boot, such as an object store or an internal artifact server.
- Outbound access from new hosts to your Fleet server and to `updates.fleetdm.com` (443).
- A Fleet server whose certificate the host trusts. If you use a private CA, see the note in step 2.
- [fleetctl](https://fleetdm.com/guides/fleetctl) on your workstation.

> **Note:** Assigning hosts to a specific fleet by enroll secret requires Fleet Premium. Without it, hosts enroll with the global secret.

## Step 1: Build and host a package without a secret

Build the package without `--fleet-url` or `--enroll-secret`. cloud-init supplies both at first boot, so one package works for every fleet and you never store a secret in your artifact host.

```sh
fleetctl package --type=deb --enable-scripts
fleetctl package --type=rpm --enable-scripts
```

Add `--arch=arm64` for arm64 hosts. Upload the package to your artifact host, then record its SHA-256:

```sh
shasum -a 256 fleet-osquery_1.61.0_amd64.deb
```

fleetd updates itself after install, but new hosts start on the version you built. Rebuild the package on a regular schedule so new hosts don't download a large update on first boot.

## Step 2: Write the user-data

This `#cloud-config` writes the Fleet URL and enroll secret, installs the package, and waits for enrollment. Replace the placeholders.

```yaml
#cloud-config
write_files:
  - path: /etc/fleet/enroll.env
    permissions: "0600"
    content: |
      ORBIT_FLEET_URL=https://fleet.example.com
      ORBIT_ENROLL_SECRET=YOUR_ENROLL_SECRET
  - path: /etc/systemd/system/orbit.service.d/10-fleet-enroll.conf
    permissions: "0644"
    content: |
      [Service]
      EnvironmentFile=/etc/fleet/enroll.env
runcmd:
  - |
    set -eu
    pkg=/var/tmp/fleetd.deb
    curl -fsSL --retry 5 --retry-connrefused -o "$pkg" https://packages.example.com/fleet/fleet-osquery_1.61.0_amd64.deb
    echo "YOUR_PACKAGE_SHA256  $pkg" | sha256sum -c -
    dpkg -i "$pkg"
    rm -f "$pkg"
    for i in $(seq 1 45); do [ -s /opt/orbit/secret-orbit-node-key.txt ] && exit 0; sleep 2; done
    echo "fleetd did not enroll within 90s" >&2
    exit 1
```

For RPM-based distributions, change the file name to `fleetd.rpm`, the URL to your `.rpm`, and `dpkg -i "$pkg"` to `dnf install -y "$pkg"`. AlmaLinux 10 was tested. Other `dnf`-based distributions use the same steps.

How this works:

- `write_files` runs before `runcmd`. The drop-in adds your file to the `orbit` service's `EnvironmentFile` list, and later files win, so the values override the package's defaults. Fleet's package doesn't mark `/etc/default/orbit` as a config file, so editing it directly is overwritten on reinstall.
- The package starts the service as soon as it installs, so fleetd starts with the right values on its first run.
- The last loop is a gate. fleetd writes `/opt/orbit/secret-orbit-node-key.txt` after it enrolls. If that never happens, cloud-init reports `status: error` instead of finishing quietly.

> **Warning:** Keep all commands in one `runcmd` script that starts with `set -eu`. cloud-init writes separate `runcmd` items as separate lines with no `set -e`, and only the last command's exit code counts. With one item per command, a failed `sha256sum` check is ignored and the package installs anyway.

> **Note:** On Debian and Ubuntu, cloud-init's `packages:` key can't install a `.deb` from a URL, and it reports an error. On RPM-based distributions it works, but it doesn't verify a checksum.

To trust a private CA, add the CA certificate with `write_files` at `/etc/fleet/fleet-ca.pem` and add `ORBIT_FLEET_CERTIFICATE=/etc/fleet/fleet-ca.pem` to `enroll.env`.

You can set any other fleetd option the same way in `enroll.env`. For example, `ORBIT_HOST_IDENTIFIER=instance` (see [Cloned images and duplicate hosts](#cloned-images-and-duplicate-hosts)), `ORBIT_END_USER_EMAIL`, and `ORBIT_ENABLE_SCRIPTS`.

## Step 3: Validate the user-data

Check the file against cloud-init's schema before you use it. It catches misspelled keys and wrong types.

```sh
cloud-init schema --config-file user-data.yaml
```

This needs cloud-init installed, so run it on any Linux machine or a test VM.

## Step 4: Deliver the user-data

How you hand over user-data depends on the platform:

- **Cloud providers:** paste the file into the instance's user-data field, or set it in Terraform or your provisioning tool. AWS limits user-data to 16 KB.
- **Local VMs and bare metal without a metadata service:** create a volume labeled `CIDATA` containing `user-data` and `meta-data` files and attach it. `meta-data` must contain an `instance-id`, and cloud-init treats a new `instance-id` as a new machine.
- **Machines you control the boot of:** cloud-init also accepts `ds=nocloud;s=https://your-server/path/` on the kernel command line or in the SMBIOS serial number, and reads `user-data`, `meta-data`, and `vendor-data` from that URL.

## Step 5: Verify

On the new host, wait for cloud-init to finish:

```sh
cloud-init status --wait --long
```

`status: done` means the package installed and fleetd enrolled. In Fleet, go to **Hosts** and confirm the host appears in the fleet that matches its enroll secret.

## Physical machines with Ubuntu autoinstall

For laptops and desktops, Ubuntu's unattended installer passes user-data to the installed system's first boot. Put the cloud-config from step 2 under `autoinstall.user-data`:

```yaml
#cloud-config
autoinstall:
  version: 1
  locale: en_US.UTF-8
  keyboard:
    layout: us
  identity:
    hostname: workstation
    username: ubuntu
    password: "YOUR_PASSWORD_HASH"
  ssh:
    install-server: true
  storage:
    layout:
      name: direct
  user-data:
    write_files:
      # same write_files entries as step 2
    runcmd:
      # same runcmd script as step 2
```

Provide it as a `CIDATA` volume, and add `autoinstall` to the installer's kernel command line. Without it, the installer asks for confirmation before erasing the disk.

This was validated with Ubuntu Server 24.04. The installed system enrolled 25 seconds after its first boot, and cloud-init then disabled itself. It wasn't tested with the Ubuntu Desktop installer.

## Protect the enroll secret

An enroll secret lets anyone enroll a host into that fleet. It doesn't expire, can't be single-use, and isn't rate limited. cloud-init isn't built to keep it private:

- On clouds, any process on the instance can read user-data from the metadata service.
- On the host, cloud-init keeps a root-only copy of user-data in `/var/lib/cloud`. After the autoinstall path, copies also exist in `/etc/cloud/cloud.cfg.d/99-installer.cfg` and `/var/log/installer/autoinstall-user-data`.
- Deleting those copies doesn't help. cloud-init recreates them on every boot while the datasource still supplies the data.

Reduce the risk:

- Use a dedicated fleet and enroll secret for hosts provisioned this way, and rotate it on a schedule. Fleet accepts several secrets at once, so add the new one before removing the old one. Hosts that are already enrolled aren't affected.
- Fetch the secret from a secrets manager at boot instead of putting it in user-data. Do this in `runcmd`, and don't run the script with `bash -x`, which prints secrets.
- Attach the `CIDATA` volume only for first boot, so the secret isn't left on removable media. The copies cloud-init already made on disk remain.

## Cloned images and duplicate hosts

Fleet identifies a host by its hardware UUID. Two problems follow:

- **Images that already contain an enrolled fleetd.** Clones share the node key, so Fleet shows several machines as one host, and that host's name and details flip between them. There's no error. Install fleetd at first boot instead. If you must bake it into an image, stop `orbit` and delete `/opt/orbit/secret-orbit-node-key.txt`, `/opt/orbit/osquery.db`, `/opt/orbit/identifier`, and `/opt/orbit/setup_experience.json` before you capture the image.
- **Hypervisors that give several VMs the same hardware UUID.** With the default identifier, these VMs keep overwriting each other and only one host stays in Fleet. Add `ORBIT_HOST_IDENTIFIER=instance` to `enroll.env` so each install gets its own identity. The trade-off is that reinstalling fleetd creates a new host record.

Cloud instances that are destroyed leave offline hosts behind. Turn on [host expiry](https://fleetdm.com/docs/configuration/yaml-files) to remove them. The shortest window is one day.

## Troubleshooting

**`cloud-init status` shows `error` and the host never enrolled.** Read `/var/log/cloud-init-output.log`. A `FAILED` line from `sha256sum` means the package doesn't match the checksum. The line `fleetd did not enroll within 90s` means the package installed but enrollment didn't complete. Run `journalctl -u orbit` on the host.

**The `orbit` log repeats `enroll request: unauthenticated, or invalid token`.** The enroll secret is wrong or was rotated out. Fix `/etc/fleet/enroll.env` and run `systemctl restart orbit`.

**The `orbit` log repeats `connection refused` or a timeout.** The host can't reach your Fleet server. fleetd keeps retrying, and it enrolled within seconds of Fleet coming back in testing, so no action is needed once the network is fixed.

**The `orbit` log repeats `end user authentication required` and `no user session found`.** The fleet requires end user authentication, which needs a person to log in through a browser. Add `ORBIT_BYPASS_END_USER_AUTH=true` to `enroll.env` for headless hosts, or use a fleet that doesn't require it.

<meta name="articleTitle" value="Enroll Linux hosts on first boot with cloud-init">
<meta name="authorFullName" value="Allen Houchins">
<meta name="authorGitHubUsername" value="allenhouchins">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-09-29">
<meta name="description" value="Install fleetd at first boot with cloud-init so new Linux VMs and Ubuntu installs enroll in Fleet automatically, with no one touching them.">

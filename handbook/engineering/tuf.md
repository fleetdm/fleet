# The Update Framework (TUF)

This handbook page outlines the processes required to create and maintain a TUF repo at Fleet. 

## Create a new TUF repo

> This process requires use of the `fleetctl` binary on Ubuntu. As of Nov. 6, 2024, Fleet only builds a `fleetctl` binary for Linux on x64. For this reason, a VM of Ubuntu on a newer Silicon MacBook (ARM) will not work. A device with an x64 processor is required. 

1. Follow the guide to create a [bootable Ubuntu USB drive](https://ubuntu.com/tutorials/create-a-usb-stick-on-ubuntu#1-overview) running the latest LTS version of Ubuntu Desktop. 

2. Download the latest version of the `fleetctl` for Linux from the the [Fleet releases GitHub page](https://github.com/fleetdm/fleet/releases).

3. Download the `tuf` CLI from [go-tuf's v0.5.2 GitHub releases page](https://github.com/theupdateframework/go-tuf/releases/tag/v0.5.2). It's important to use the same version of `go-tuf` that is used by `fleetctl`.

4. Connect a new USB drive and copy the `fleetctl` binary and `tuf` binary to the USB drive. The `tuf` binary will likely be in `~/go/bin/`.

5. Open 1Password and click "New item", then select "Secure note". Name the note to match the new TUF repo. 

6. Next, generate four passwords, one for each role's key. Click "Add more", select "Password", then "Generate a new password". 

7. Click the "Password" label in the input field and change it to "root passphrase". Repeat this two two more times for "root2 passphrase" and "root3 passphrase" as backups. 

8. Repeat three more times for "targets passphrase", "snapshot passphrase", and "timestamp passphrase". Backups are not necessary for these keys because root keys can generate new ones. 

9. Disconnect both USB drives. 

10. Connect the bootable Ubuntu USB drive to the signing device and boot to Ubuntu. When the boot screen appears, press the key the manufacturer has set to enter the boot menu. This is typically F1, F10, or ESC.

11. On the boot menu, select the Ubuntu USB drive, then "Try or Install Ubuntu" to boot directly from the USB drive. 

12. Walk through the setup steps and **do not** connect to the internet. 

13. After reaching the Ubuntu desktop, plug in the USB drive containing the `fleetctl` and `tuf` binaries. 

14. Click the "Show Apps" icon in the bottom-left corner, and open the Terminal app. 

15. Mount the USB drive and navigate to the directory. 

16. Run `./fleetctl updates init` to initialize a new TUF repo on the USB drive. Manually type in the passphrases for each role's key that you generated in 1Password. 

17. Create multiple root keys in case one is lost. Run `mv keys/root.json keys/root1.json` to retain the first root key. Then run `./tuf gen-key root` and enter the passphrase for "root2". Repeat one more time for "root3". When complete, you should have three root keys: `root1.json`, `root2.json`, `root3.json`. 

18. The last root key generated (`root3.json`) will be the only signature on the metadata at `staged/root.json`. We want to sign with all root keys. Run `mv keys/root1.json keys/root.json`, then run `./tuf sign root.json` to sign with key 1. Repeat the step for key 2 so that your `staged/root.json` is signed by all three root keys. 

19. Plug in additional USB drives and copy only the `keys` directory. They will serve USB root backups. 

20. Next, plug in a USB drive to serve as the repo drive to copy files for signing. This USB drive will never contain keys. When plugged in, copy only the `/repository` and `/staged` directories. Make sure **not** to copy the `/keys` directory. 

21. Next, plug in a last USB drive to serve as your day-to-day signing drive. This will contains the targets, snapshot, and timestamp keys, but will not contain the root keys. Copy the `repository` and `staged` directories. Next, copy only the `keys/targets.json`, `keys/snapshot.json`, and `keys/timestamp.json` keys to the drive. Do **NOT** copy any of the root keys. 

22. At this point, all USB drives can be removed and your offline signing device or VM turned off. 

23. On your device connected to the internet, plug in the repo USB drive. This one should contain only the `repository` and `staged` directories. Copy the files from the USB drive to a working directory on your internet-connective device. 

24. Upload the files to your desired file hosting location, typically AWS S3 or CloudFlare R2. 

You now have a functional, secure TUF repo. You can now configure and use the [Fleet TUF repo release script](https://github.com/fleetdm/fleet/tree/main/tools/tuf) to add new file targets. 

If you need to run TUF commands that are not available using the `fleetctl` binary, additional functionality is available using the `tuf` binary [documented by go-tuf](https://pkg.go.dev/github.com/theupdateframework/go-tuf#section-readme).

## Read and write to TUF repo on Cloudflare R2

Fleet hosts our TUF repo in Cloudflare R2 buckets for production and staging, updates.fleetdm.com and updates-staging.fleetdm.com. Read and write operations are performed used the [AWS CLI](https://developers.cloudflare.com/r2/examples/aws/aws-cli/) tool configured to communicate with R2.

Once configured, use the [Fleet TUF repo release script](https://github.com/fleetdm/fleet/tree/main/tools/tuf) to add new file targets.  You can use the `aws s3 cp` command to push and pull objects:  `aws s3 cp . s3://<bucket-name> --recursive --endpoint-url https://<accountid>.r2.cloudflarestorage.com`

## Add new TUF keys for authorized team members

The CTO is responsible for determining who has access to push agent updates. Timestamp and Snapshot keys can be held online, so their use can be automated, but Targets and Root keys must always be held offline. The root keys are held by the CTO and CEO in secure locations. Root keys are retrieved once per year to rotate them before their annual expiration, or to sign for new Targets keys as needed. Targets keys may be generated to provide approved team members the ability to push agent updates to the TUF repo. 

This process requires running TUF commands that are not available using the `fleetctl` binary, so the `tuf` CLI binary [documented by go-tuf](https://pkg.go.dev/github.com/theupdateframework/go-tuf#section-readme) needs to be downloaded and compiled for local use.

There are two roles required to complete these steps, the "Root" role who holds the root keys, and the "Releaser" role, who is gaining access to push updates. 

1. The Releaser creates a new local directory to store the TUF repo. The Releaser creates a sub-directory called `repository`.

2. The Realeaser pulls down the contents of the TUF repo into the `repository` sub-directory. 

3. From the root of their TUF directory, the Releaser runs `tuf gen-key targets`. This will create a `keys` sub-directory and `staged` sub-directory.  Next, the Releaser runs `tuf gen-key snapshot`, then `tuf gen-key timestamp` to create keys for those roles. 

4. The Releaser copies the `keys` directory to a USB drive, and deletes the `keys` directory from their local hard drive. 

5. The Releaser sends the `staged/root.json` to the Root role for signing. Note this file is safe to share and is publicly available. 

6. The Root role receives the `staged/root.json` file and copies it to a USB drive. 

7. The Root role boots into the secure Ubuntu boot drive created during TUF repo creation. 

8. The Root role connects the USB drive containing the `staged/root.json` file for signing. 

9. The Root role connects the USB drive containing the root keys. 

10. The Root role copies the `staged/root.json` onto the root keys USB at `staged/root.json`. 

11. The root keys USB contains the `tuf` binary. Run `./tuf sign root.json` to sign the staged root metadata. 

12. The Root role copies the signed `staged/root.json` back to the original USB drive they copied it from. 

13. The Root role turns off the Ubuntu boot drive and accesses an online computer. 

14. The Root role connects the USB drive containing the signed `staged/root.json` file and copies it to their local hard drive's TUF location in the same `staged/root.json`. 

15. From the root of their local TUF repo, the Root role runs `tuf commit` to commit the staged root metadata to the `repository` directory. 

16. The Root role pushes the updated contents of the `repository` directory to the remote TUF server. 

17. The Releaser role can now run `tuf sign` to sign agent updates using their offline Targets key.

## Rotate the root keys 

The root keys expire every year and must be manually rotated at least 30 days prior to expiration. 

1. The root keys are retrieved from their secure location. 

2. The offline Ubuntu bootable USB drive is turned on. 

3. The root keys USB drive is connected to the Ubuntu bootable instance. Before proceeding, make two backups of the root keys on USB drives for safe keeping. They will be deleted when the root keys have been successfully rotated. 

4. Add three new root keys using the steps documented in creating a new TUF repo. 

5. Run `tuf sign root.json` to sign the newly added root keys with an existing root key. 

6. Run `tuf commit` to commit the staged metadata with new root keys. 

7. Using one of the new root keys, run `tuf revoke-key <role> <id>`. Run this command for each of the old, expiring root keys. 

8. Using each of the new root keys, run `tuf sign root.json` to sign the root metadata removing the old root keys and adding the new keys so that the new root.json is signed by all root keys.

9. Using one of the new root keys, run `tuf commit` to commit the staged root metadata. 

10. Confirm the file in `repository/root.json` contains the new root key ids by comparing the ids listed in `signed.roles.root.keyids` to the signatures in `signatures`. Make sure all root ids have signed.

11. Copy the `repository` directory to the local drive of an online device and push to the remote TUF repo. 

12. Confirm that agent updates are continuing with the new `root.json`. Once confirmed, it is safe to delete the old root keys and backup the new keys.

<meta name="maintainedBy" value="lukeheath">
<meta name="description" value="This page outlines our TUF creation and maintenance processes.">
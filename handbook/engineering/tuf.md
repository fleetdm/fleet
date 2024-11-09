# The Update Framework (TUF)

This handbook page outlines the processes required to create and maintain a TUF repo at Fleet. 

## Create a new TUF repo

> This process requires use of the `fleetctl` binary on Ubuntu. As of Nov. 6, 2024, Fleet only builds a `fleetctl` binary for Linux on x64. For this reason, a VM of Ubuntu on a newer Silicon MacBook (ARM) will not work. A device with an x64 processor is required. 

1. Follow the guide to create a [bootable Ubuntu USB drive](https://ubuntu.com/tutorials/create-a-usb-stick-on-ubuntu#1-overview) running the latest LTS version of Ubuntu Desktop. 

2. Download the latest version of the `fleetctl` for Linux from the the [Fleet releases GitHub page](https://github.com/fleetdm/fleet/releases).

3. Download the `tuf` CLI from [go-tuf's v0.7.0 GitHub releases page](https://github.com/theupdateframework/go-tuf/releases/tag/v0.7.0).

4. Connect a new USB drive and copy the `fleetctl` binary and `tuf` binary to the USB drive. The `tuf` binary will likely be in `~/go/bin/`.

5. Open 1Password and click "New item", then select "Secure note". Name the note to match the new TUF repo. 

6. Next, generate four passwords, one for each role's key. Click "Add more", select "Password", then "Generate a new password". 

7. Click the "Password" label in the input field and change it to "root passphrase". Repeat this two two more times for "root2 passphrase" and "root3 passphrase" as backups. 

8. Repeat three more times for "targets passphrase", "snapshot passphrase", and "timestamp passphrase". Backups are not necessary for these keys because root keys can generate new ones. 

9. Disconnect both USB drives. 

10. Connect the bootable Ubuntu USB drive and restart your computer. When the boot screen appears, press the key the manufacturuer has set to enter the boot menu. This is typically F1, F10, or ESC.

11. On the boot menu, select the Ubuntu USB drive, then "Try or Install Ubuntu" to boot directly from the USB drive. 

12. Walk through the setup steps and **do not** connect to the internet. 

13. After reaching the Ubuntu desktop, plug in the USB drive containing the `fleetctl` and `tuf` binaries. 

14. Click the "Show Apps" icon in the bottom-left corner, and open the Terminal app. 

15. Mount the USB drive and navigate to the directory. 

16. Run `./fleetctl updates init` to initialize a new TUF repo on the USB drive. Manually type in the passphrases for each role's key that you generated in 1Password. 

17. Create multiple root keys in case one is lost. Run `mv keys/root.json keys/root1.json` to retain the first root key. Then run `./tuf gen-key root` and enter the passphrase for "root2". Repeat one more itme for "root3". When complete, you should have three root keys: `root1.json`, `root2.json`, `root3.json`. 

18. The last root key generated (`root3.json`) will be the only signatured on the file at `staged/root.json`. We want to sign with all root keys. Run `mv keys/root1.json keys/root.json`, then run `./tuf sign root.json` to sign with key 1. Repeat the step for key 2 so that your `staged/root.json` is signed by all three root keys. 

19. Plug in additional USB drives and copy only the `keys` directory. They will serve USB root backups. 

20. Next, plug in a USB drive to serve as the repo drive to copy files for signing. This USB drive will never contain keys. When plugged in, copy only the `/repository` and `/staged` directories. Make sure **not** to copy the `/keys` directory. 

21. Next, plug in a last USB drive to serve as your day-to-day signing drive. This will contains the targets, snapshot, and timestamp keys, but will not contain the root keys. Copy the `repository` and `staged` directories. Next, copy only the `keys/targets.json`, `keys/snapshot.json`, and `keys/timestamp.json` keys to the drive. Do **NOT** copy any of the root keys. 

22. At this point, all USB drives can be removed and your offline signing device or VM turned off. 

23. On your laptop connected to the internet, plug in the repo USB drive. This one should contain only the `repository` and `staged` directories. Copy the files from the USB drive to a working directory on your internet-connective device. 

24. Upload the files to your desired file hosting location, typically AWS S3 or CloudFlare R2. 

You now have a functional, secure TUF repo. You can now configure and use the [Fleet TUF repo release script](https://github.com/fleetdm/fleet/tree/main/tools/tuf) to add new file targets. 

If you need to run TUF commands that are not available using the `fleetctl` binary, additional functionality is available using the `tuf` binary [documented by go-tuf](https://pkg.go.dev/github.com/theupdateframework/go-tuf#section-readme).




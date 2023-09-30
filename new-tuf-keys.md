
Perform the following operations on a USB stick so that the keys are not available on the local filesystem (only available when the USB is connected).

1. `cd` to a directory on the USB stick.

2. Generate a new set of keys with `fleetctl updates init`.
   
   - The `root` key will not be used, so any password can be provided.
   - For each other key (`targets`, `timestamp`, `snapshot`), generate a unique, strong password and save it in 1Password.
   
3. Share the generated `root.json` file that includes the key IDs and public keys for all of the generated keys.

Now, update the root metadata 

`tuf add-key targets <public key>`
`tuf add-key snapshot <public key>`
`tuf add-key timestamp <public key>`

(these steps do not require signing so can be done online without root keys inserted)

Manually edit the expiration timestamp to 1 year out (or you can use the `--expires` option when running the `add-key` command`)

Now take the updated `staged/root.json` and bring it to an offline device (live Linux on USB stick) with the root keys inserted. Sign the root metadata with `tuf sign root.json`, then commit with `tuf commit`. This will create identical files `repository/root.json` and `repository/<n>.root.json` (with `n` being the next incrementing integer in the sequence of root files). Copy each of these onto both the USB containing the root keys as well as the USB containing the live repository.

Back in an online environment, the live repository metadata with the 2 new root metadata files can be uploaded to the S3 bucket.

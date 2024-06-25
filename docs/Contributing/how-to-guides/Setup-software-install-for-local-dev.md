# Setup software install for local dev

If you are working on a feature for adding, removing, or installing software installers
you'll likely want to upload software installers. This guide walks you through the
steps needed to be able to upload software installers while working in you
local dev environment.

## Steps to set up

1. Ensure you are running your fleet server with the `--dev` flag. This will
ensure that your local server is started with default values needed for uploading
software installers. [Here are
those values](https://github.com/fleetdm/fleet/blob/4180ec83a286e9679abddf9b595eeacd96a031d7/cmd/fleet/main.go#L85-L90),
for those who are curious.

2. Go to http://localhost:9001. You will see a Minio GUI there to login.

3. Login with the credentials username: `minio` and password: `minio123!`

4. Click on the `Create Bucket` button.

5. Name the new bucket `software-installers-dev` in the `Bucket Name` input. You
**must** name your bucket this so that it matches the bucket name value when
running the fleet server with `--dev`.

6. Click the `Create Bucket` button. The bucket should now appear on your
`Buckets` page.

You should now be able to upload software installers. They will be added into
this bucket you just created.

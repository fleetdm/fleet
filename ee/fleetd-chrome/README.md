# Fleetd Chrome Extension

## Packaging the extension locally
Generate a .pem file to be the key for the chrome extension.

(In parent dir)
Run the following command to generate an extension.

``` sh
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --pack-extension=./fleetd-chrome --pack-extension-key=path/to/chrome.pem
```

## Adding Chrome to Fleet
To learn how to package and add hosts to Fleet, visit: https://fleetdm.com/docs/using-fleet/enroll-hosts#enroll-chromebooks.

## Debugging

### Service worker

View service worker logs in chrome://serviceworker-internals/?devtools (in production), or in chrome://extensions (only during development).

### Manual Enroll

> Steps 1 and 2 can be performed on your workstation. Step 3 and 4 are to be executed on the target Chromebook.

1. Create your .env file:

> IMPORTANT: The address in `FLEET_URL` must have a valid TLS certificate.

```sh
echo 'FLEET_URL="https://your-fleet-server.example.com"' >> .env
echo 'FLEET_ENROLL_SECRET="<your enroll secret>"' >> .env
```

To test with your local Fleet server, you can use [Tunnelmole](https://github.com/robbie-cahill/tunnelmole-client) or [ngrok](https://ngrok.com/).


Tunnelmole:

```sh
tmole 8080
```

ngrok:

```sh
ngrok http https://localhost:8080
```

2. Build the "unpacked extension":
```sh
npm install && npm run build
```
The above command will generate an unpacked extension in `./dist`.

3. Send the `./dist` folder to the target Chromebook.

4. In the target Chromebook, go to `chrome://extensions`, toggle `Developer mode` and click on `Load unpacked` and select the `dist` folder.

## Testing

### Run tests

```sh
npm run test
```

## Release

1. After your changes have been merged to the main branch, create a new branch for the release.
2. At the top of the repo, update CHANGELOG.md by running `version="X.X.X" make changelog-chrome`
3. Review CHANGELOG.md
4. At `ee/fleetd-chrome`, run `npm version X.X.X` to update the version in `package.json` and `package-lock.json`
5. Commit the changes and tag the commit with `fleetd-chrome-vX.X.X-beta`. This will trigger the beta release workflow.
6. Test your beta release:
   1. Open the Google admin console (https://admin.google.com)
   2. Go to Devices > Chrome > Apps & Extensions > Users & browsers
   3. Under Organizational Units, select the group that your ChromeOS device is in, or the top-level Fleet Device Management OU to test the beta on all ChromeOS devices (yours may not be in a specific OU).
   4. Select the production extension (fleeedmmihkfkeemmipgmhhjemlljidg), change its installation policy to "Block", and save your changes. This will remove the production extension from the selected devices so that you can test the beta.
   5. Select the beta extension (bfleegjcoffelppfmadimianphbcdjkb), change its installation policy to "Force install" and save your change. This will push the beta extension out to the selected devices.
   6. Verify that the beta extension has installed on a device using the Chrome extension manager, and test your changes!
7. Once the beta release is tested, make a PR with the updates to the version and changelog and tag the commit with `fleetd-chrome-vX.X.X`. This will trigger the release workflow. 
8. In the Google admin console, set the beta extension installation policy to "Block" and the production extension to "Force install".
9. Announce the release in the #help-engineering channel in Slack.

Using GitHub Actions, the build is automatically uploaded to R2 and properly configured clients should be able to update immediately when the job completes. Note that automatic updates seem to only happen about once a day in Chrome -- Hit the "Update" button in `chrome://extensions` to trigger the update manually.

### Beta releases

Beta releases are pushed to `https://chrome-beta.fleetdm.com/updates.xml` with the extension ID `bfleegjcoffelppfmadimianphbcdjkb`.

Kick off a beta release by updating the [package.json](./package.json), then tag a commit with `fleetd-chrome-vX.X.X-beta` to kick off the build and deploy.

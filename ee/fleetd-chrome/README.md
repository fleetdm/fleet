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
   1. In Google Groups (https://groups.google.com), switch to "All groups" and remove yourself from the "Chromebooks" group. You can only be in one of "Chromebooks" or "ChromeOS testers" at a time.
   2. Add yourself to the "ChromeOS testers" group. Members of this group get the beta extension (bfleegjcoffelppfmadimianphbcdjkb).
   3. Verify that the beta extension has installed on your device using the Chrome extension manager, and test your changes!
7. Once the beta release is tested, make a PR with the updates to the version and changelog and tag the commit with `fleetd-chrome-vX.X.X`. This will trigger the release workflow. 
8. Remove yourself from the "ChromeOS testers" group. To go back to the production extension (fleeedmmihkfkeemmipgmhhjemlljidg), add yourself to the "Chromebooks" group.
9. Announce the release in the #help-releases and #help-engineering channels in Slack.

Using GitHub Actions, the build is automatically uploaded to R2 and properly configured clients should be able to update immediately when the job completes. Note that automatic updates seem to only happen about once a day in Chrome -- Hit the "Update" button in `chrome://extensions` to trigger the update manually.

## Troubleshooting

The first step in troubleshooting a released extension (beta or production) is to go to [chrome://extensions](chrome://extensions) in your browser (or click the "Manage Extensions" link under the extensions menu), turn on Developer Mode using the toggle at the top-right corner of the page, then click the "Inspect views service worker" link in the Fleetd for Chrome extension. From there you can view the Console to see any error messages that the extension is logging.

### Chrome device not appearing in Dogfood

If the extension is logging errors like `enroll failed: no matching secret found`, check the enroll secret set in the "Policy for extensions" for the extension in the Google admin console (Apps & Extensions -> Users & Browsers -> select the fleet extension -> scroll all the way down to the bottom of the right-hand drawer). Ensure that the `enroll_secret` value represents a valid secret for a fleet on the Fleet dogfood instance.  

### Device not receiving updated extension version

First, try hitting the "Update" button on the [chrome://extensions](chrome://extensions) page in your browser. If that doesn't work, you can toggle the policy for the extension to "Block", wait a moment for it to be removed from the device, and then change it back to "Force install". 
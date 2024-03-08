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

Release a new version via GitHub automation. Update the [package.json](./package.json) and [updates.xml](./updates.xml) versions, then tag a commit with `fleetd-chrome-vX.X.X` to kick off the build and deploy. The build is automatically uploaded to R2 and properly configured clients should be able to update immediately when the job completes. Note that automatic updates seem to only happen about once a day in Chrome -- Hit the "Update" button in `chrome://extensions` to trigger the update manually.

### Beta releases

Beta releases are pushed to `https://chrome-beta.fleetdm.com/updates.xml` with the extension ID `bfleegjcoffelppfmadimianphbcdjkb`.

Kick off a beta release by updating the [package.json](./package.json) and [updates-beta.xml](./updates-beta.xml) versions, then tag a commit with `fleetd-chrome-vX.X.X-beta` to kick off the build and deploy.

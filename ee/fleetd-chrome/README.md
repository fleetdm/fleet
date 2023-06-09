# Fleetd Chrome Extension

## Packaging the extension
Generate a .pem file to be the key for the chrome extension.

(In parent dir)
Run the following command to generate an extension.

``` sh
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --pack-extension=./fleetd-chrome --pack-extension-key=path/to/chrome.pem
```

## Adding Chrome to Fleet
To learn how to package and add hosts to Fleet, visit: https://fleetdm.com/docs/using-fleet/adding-hosts#add-chromebooks-with-the-fleetd-chrome-extension.

## Debugging

### Service worker

View service worker logs in chrome://serviceworker-internals/?devtools (in production), or in chrome://extensions (only during development).

### Dev

1. Create your .env file:
```
echo 'FLEET_URL="<some_url>"' >> .env
echo 'FLEET_ENROLL_SECRET="<your enroll secret>"' >> .env
```
2. Build:
```
npm install && npm run build
```
3. The unpacked extension is in the `dist` dir.

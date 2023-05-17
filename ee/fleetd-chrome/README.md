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
To learn how to debug the Fleetd Chrome extension, visit: https://fleetdm.com/docs/contributing/testing-and-local-development#fleetd-chrome-extension.

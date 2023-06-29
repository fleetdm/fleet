# Deploying ChromeOS test Extensions to enrolled ChromeBooks

As part of validating any ChromeOS extention, run this process to force-install extension on our debug Chromebooks.

## Build the extension

### Bump the extension version

Open ee/fleetd-chrome package.json for edit.
Modify the version field at the top of the file (Line 4 when writing this doc)
```
{
  "name": "fleetd-for-chrome",
  "description": "Extension for Fleetd on ChromeOS",
  "version": "1.0.3",
  ...
```

### Build the distribution folder

```
cd ee/fleetd-chrome
yarn run build
```

### Pack the extension

-Go to chrome web browser.
-go to chrome://extensions
-Press "Pack Extensions" button (Top left of the screen)
-Press the top Browse button and select the newly created dist folder (ee/fleetd-chrome/dist)
-Press "Pack Extension" (No need to give it a key. It will generate it)

### Load the new extension to the Chrome web browser

-Open the finder app 
-Drag and srop the ee/fleetd-chrome/dist.crx binary file to chrome web browser
-Press "Add Extension"
-Verify that the extension works
-**Copy the appid for later use.**

## Run a local server to make the new extension available

### Create the server

```
cd ee/fleetd-chrome
python3 -m http.server
```
Verify that it works by going to http://localhost:8000 and see the files.



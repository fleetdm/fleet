# Deploying ChromeOS test extensions to enrolled Chromebooks

As part of validating any ChromeOS extension, run this process to force-install the extension on Chromebooks for debugging.

## Build the extension

### Bump the extension version

Modify the version field at the top of the [`package.json`](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/package.json) file in `ee/fleetd-chrome`

Update the version in [`updates.xml`](https://github.com/fleetdm/fleet/blob/main/ee/fleetd-chrome/updates.xml) to match the `package.json` version.

### Build the distribution folder

```sh
cd ee/fleetd-chrome
yarn run build
```

### Pack the extension

Navigate to chrome://extensions in your Chrome web browser.
- In developer mode, select "Pack extension"
- Set "Extension root directory" to the newly-created `ee/fleetd-chrome/dist` folder
- Press "Pack extension" (key name will auto-generate)

### Load the new extension to the Chrome web browser

- Open the finder app 
- Drag and drop the `ee/fleetd-chrome/dist.crx` binary file on top of a Chrome web browser window
- Press "Add Extension"
- Verify that the extension works
- **Copy the `appid` for later use**

## Run a local server to make the new extension available

### Edit update.xml
Open `ee/fleetd-chrome/update.xml` in your text editor and modify:
- The version.
- The `appid` (copied previously). This will only be done for debug versions. For production, we will keep the original ID we have.

### Create the server

```sh
cd ee/fleetd-chrome
python3 -m http.server
```
- Verify that it works by going to http://localhost:8000 to see the files.

```sh
cd ee/fleetd-chrome
npm install -g localtunnel
lt --port 8000 --subdomain test-new-tables
```
- In your web browser go to: http://test-new-tables.loca.lt
- Click the hazard link on item number 1 (below the big button "Click To Submit"). From the new page, copy the IP and paste it into the previous page in the window.
- Open `ee/fleetd-chrome/update.xml` in your text editor and modify the codebase to use the newly created URL (in this example: http://test-new-tables.loca.lt/dist.crx).

### Deploy the extension using Google Admin

> Follow the instructions [here](https://fleetdm.com/docs/using-fleet/enroll-hosts#enroll-chromebooks) for installing the fleetd Chrome extension, with the following modifications:
> + Select the "ChromeOSTesting" group.
> + For "Extension ID", use the ID previously copied.
> + For "Installation URL", use `http://test-new-tables.loca.lt/updates.xml`.
> + Remove the filters (the filters with our `appid`).
> + For "Policy for extensions", copy over the JSON from the original extension.

<meta name="pageOrderInSection" value="750">

# Fleetd Chrome Extension

## Packaging the extension
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

[Tunnelmole](https://github.com/robbie-cahill/tunnelmole-client) is an open source tunnelling tool that readily provides a Public URL that forwards traffic to your local machine through a secure tunnel. To install Tunnelmole, use the following for Linux, Mac, and Windows Subsystem for Linux:

```sh
curl -O https://tunnelmole.com/sh/install.sh && sudo bash install.sh
```

For Windows without WSL, [Download tmole.exe](https://tunnelmole.com/downloads/tmole.exe) and put it somewhere in your [PATH](https://www.wikihow.com/Change-the-PATH-Environment-Variable-on-Windows).

Then, you can run Tunnelmole by typing `tmole 8080` into your terminal:

```sh
tmole 8080
```

Alternatively, [ngrok](https://ngrok.com/) is a popular closed source tunnelling tool. To test with ngrok:

```sh
ngrok http https://localhost:8080
```

2. Build the "unpacked extension":
```sh
npm install && npm run build
```
The above command will generate an unpacked extension in `./dist`.

3. Send the `./dist` folder to the target Chromebook.

4. In the target Chromebook, go to `chrome://settings`, toggle `Developer mode` and click on `Load unpacked` and select the `dist` folder.

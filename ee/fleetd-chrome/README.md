# Fleetd Chrome Extension

## Pack extension

(In parent dir)

``` sh
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --pack-extension=./fleetd-chrome --pack-extension-key=$HOME/chrome.pem
```

## Configure in Google Admin

Left menu: Devices > Chrome > Users & browsers 

Bottom right yellow + button > Add Chrome app or extension by ID

Extension ID: `fleeedmmihkfkeemmipgmhhjemlljidg`
From a custom URL: `https://chrome.fleetdm.com/updates.xml`

Then add the "Policy for extensions" to configure it:

```
{
  "fleet_url": {
    "Value": "https://fleet.example.com"
  },
  "enroll_secret":{
    "Value": "<secretgoeshere>"
  }
}
```

Select "Force install". Select "Update URL" > "Installation URL (see above)"


## Debugging

### Service worker

View service worker logs in chrome://serviceworker-internals/?devtools (in production), or in chrome://extensions (only during development).

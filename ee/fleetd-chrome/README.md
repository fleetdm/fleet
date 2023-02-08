# Fleetd Chrome Extension

## Pack extension

(In parent dir)

``` sh
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --pack-extension=./fleetd-chrome --pack-extension-key=$HOME/chrome.pem
```

## Configure in Google Admin

Left menu: Devices > Chrome > Users & browsers 

Bottom right yellow + button > Add Chrome app or extension by ID

Extension ID: `npolcaekpfbjaegcnnppmemjhadibmop` (with the current key I'm using in
development -- this ID seems to be based off the signing key)
From a custom URL: `https://crx.fleetuem.com/fleetd-chrome/updates.xml` (with the current hosting
strategy I'm using for development. Really it just needs to be URL to the `updates.xml` )

Then add the "Policy for extensions" to configure it:

```
{
  "fleet_url": {
    "Value": "http://fleet.example.com"
  },
  "enroll_secret":{
    "Value": "secretgoeshere"
  }
}
```

Select "Force install"

## SQLite

SQLite is compiled to wasm as described in https://sqlite.org/src/file/ext/wasm/. We rely on the
virtual tables work that is quite new and not yet available in the wasm version on the downloads
page.

There seemed to be a bug with the generated code that was fixed with the following patch:

```
diff ~/dev/sqlite/ext/wasm/jswasm/sqlite3.js sqlite3.js
15172c15172
<     if(!(tgt instanceof sqlite3.StructBinder.StructType)){
---
>     if (false) {
```

## Debugging

### Service worker

View service worker logs in chrome://serviceworker-internals/?devtools, or in chrome://extensions.

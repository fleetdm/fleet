# Fleetd Chrome Extension

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
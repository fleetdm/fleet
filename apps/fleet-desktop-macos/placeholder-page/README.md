# Patch notification placeholder page

A stand-in for Fleet's real My device patch notification page, for testing the
`FleetDesktop notify` toast before that page exists.

It is deliberately **not** bundled into the app. Serving it over HTTPS exercises the
same code path the production URL will use — remote load, HTTP status handling, origin
validation on the JS bridge — which a `file://` page does not.

## Deploy

Any static host works. With [Vercel Drop](https://vercel.com/drop), drag
`placeholder-page.zip` (see below) onto the page and it returns a URL.

To rebuild the zip after editing:

```bash
cd placeholder-page && zip -r ../placeholder-page.zip . -x '.*' && cd -
```

The zip has `index.html` at its root, which is what Drop expects.

## Use it

```bash
./dev-notify.sh https://your-deployment.vercel.app
```

or directly:

```bash
"build/Fleet Desktop.app/Contents/MacOS/FleetDesktop" \
    notify --url https://your-deployment.vercel.app
```

`--url` requires `https`, so a local `python3 -m http.server` won't do — it has to be
a real deployment.

## What the real page has to copy from this one

This page is the working reference for the contract. Anything the real page skips
degrades in a specific way:

| | If omitted |
|---|---|
| Transparent `body` | The page's own background covers the native card, and is the wrong colour in dark mode |
| `ready` message | The toast still appears, but only after a 1.5s grace period |
| `dismiss` / `primary` | **The toast cannot be closed** — no title bar, no close button, no Esc. Only the display timeout or `pkill` gets rid of it |
| `resize` | The card stays 525×318, clipping taller content |

The bridge, in full:

```js
window.webkit.messageHandlers.fleetDesktop.postMessage({
  v: 1,
  action: "ready" | "primary" | "dismiss" | "resize" | "log",
  payload: {}
});
```

Native tolerates unknown actions and a missing `payload`, so a newer page keeps
working against an older app.

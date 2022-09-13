# Token Rotation for Fleet Desktop

This file is based on the original proposal described at [#6348](https://github.com/fleetdm/fleet/issues/6348) modified based on a few lessons we learned and the new communication channel between Orbit and the Fleet server introduced in https://github.com/fleetdm/fleet/issues/6851

**Compatibility**

|                | Fleet Desktop < v1.0.0            | Fleet Destkop >= v1.0.0 |
| -------------- | --------------------------------- | ----------------------- |
| Server < 4.21  | OK/Rotation disabled              | OK/Rotation disabled    |
| Server >= 4.21 | Fleet Destkop breaks after 1 hour | OK/Rotation enabled     |


## Fleet Server

1. Add a new endpoint `POST /orbit/<orbit_node_key>/device_token` to create/update device tokens
    1. Add `created_at` and `updated_at` columns to the `host_device_auth` table.
    2. Do `INSERT ON DUPLICATE KEY token=token` and not `update updated_at` if the token didn't change.
1. Condsider a token expired if `now - updated_at > 1h`, APIs will return the usual authentication error when a token is expired.
1. The server doesn't need to make the `orbit_info` in the "extra detail queries" set anymore.
    
## Orbit

1. When Orbit starts
    1. If we have a token, load and verify its validity by making a request to the Fleet Server
    1. If we don't have a token or if the token is invalid, [rotate the token](#token-rotation)

1. Orbit will have two tickers running:
    - **Ticker I**: runs every 5 minutes and verifies that the current token is still valid by making a request to the Fleet Server. This is to guard against the server invalidating the token (eg: DB restored from back-up, token `updated_at` manually changed, clocks out of sync etc.) If the token is invalid, it [starts a rotation](#token-rotation).
    -  **Ticker II**: runs every 30 seconds and verifies the `mtime` of the identifier file, if `now - mtime > 1h` [starts a rotation](#token-rotation). A short interval (could be even shorter) is needed in case the computer was shut-down or went to sleep.

**Token Rotation**

To rotate a token, Orbit will generate a valid UUID and:

1. Write the value to the `identifier` file, we do this first to signal to Fleet Desktop that a rotation is happening and we can ensure it never operates on an invalid token during the exchange.
2. Do a `POST /orbit/<orbit_node_key>/device_token` with the new token value, retry three times in case of failure.

**Compatibility**

1. Keep returning `device_auth_token` in the `orbit_info` table, we might want to do this forever anyway to support live queries.
2. When Orbit starts, check if the server supports creating tokens via the API, if it doesn't:
    1. Don't do any kind of check or rotation
    2. Start a goroutine using `oklog.Group` to ping the server every 10 minutes and see if it supports token rotation. If it does, return from the group actor, which will make Orbit restart.
3. Orbit will keep sending the `FLEET_DESKTOP_DEVICE_URL` variable to accomodate old Fleet Destkop versions.

## Fleet Desktop

1. Fleet Desktop will receive the path to the identifier file as an environment variable.
2. As soon as Flet Desktop starts, it reads the identifier file and caches the `mtime` value in memory.
3. Fleet Desktop will have a ticker running every ~5 seconds to check for the `mtime` of the identifier file, if the value differs from the one stored in memory:
    1. Disable all tray items and show "Connecting..."
    2. Initialize a ticker to check for the validity of the token, enable the tray items again once we have a valid token.

**Misc**

- As soon as any request fails, disable the tray items and show "Connecting..."

## Release order

After things have been tested on unstable channels and Dogfood, it's important to release in the following order:

1. Orbit to the stable channel
2. Fleet Desktop to the stable channel
3. Fleet Server

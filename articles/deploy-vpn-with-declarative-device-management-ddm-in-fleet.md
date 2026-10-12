# Deploy VPN with declarative device management (DDM) in Fleet

Apple's [`com.apple.configuration.network.vpn.ikev2`](https://developer.apple.com/documentation/devicemanagement/networkvpnikev2) declaration is one of four new VPN declaration types introduced to replace the legacy VPN payload in the `.mobileconfig` profile. Fleet supports deploying it, along with the credential asset it depends on, as of Fleet 4.90.

This guide deploys one end to end, using a shared-secret (PSK) VPN server as the example, and covers where it differs from the old `.mobileconfig` VPN payload.

## How is this different?

The old VPN payload embedded the shared secret or credentials directly in the profile. DDM takes a cleaner approach: every VPN declaration type separates configuration (the declaration) from credentials (a separate DDM asset), and the asset itself doesn't carry the secret value either. It carries a URL, and the device fetches the credential content from that URL directly. Fleet supports referencing host attributes like the hardware serial number or IdP identity directly inside a DDM declaration, but a VPN's shared secret works differently: it lives at infrastructure you host, reachable over HTTPS independently of Fleet.

## Requirements

- macOS 27+ on the host. Earlier macOS versions don't recognize VPN declarations and will reject any profiles with an `Error.UnknownDeclarationType` message.
- Fleet Premium.
- An HTTPS endpoint with a certificate the device already trusts, to host the credential content.

## How credential delivery works

A VPN declaration that authenticates with a shared secret references a separate asset declaration of type `com.apple.asset.credential.userpassword`. That asset doesn't contain the secret. It contains a `Reference.DataURL` pointing at a JSON file matching Apple's [credential reference schema](https://developer.apple.com/documentation/devicemanagement/credentials-asset), which the device fetches to resolve the actual username/password (used as the shared secret for IKEv2).

- **Fleet stores and delivers the asset JSON exactly as you upload it.** It doesn't fetch or cache your `DataURL` server-side or rewrite it, and it doesn't proxy the device's request either. The device talks to your infrastructure directly.
- **Fleet always applies MDM authentication semantics to that fetch.** Apple's spec lets an asset declare `Authentication.Type` as either `MDM` or `None`, controlling how the device authenticates when it fetches the `DataURL`. `None` is a plain, unauthenticated GET. `MDM` means the device includes the same credentials it uses to talk to its MDM server, its device-identity certificate plus any user authentication, and is willing to present them if asked. Fleet always uses `MDM` here (asset uploads don't take an `Authentication` key at all), which means you have the option to build real device-level access control into your hosting server, by requesting and validating that certificate, without any extra setup on Fleet's side. Plain HTTPS hosting also works fine if you don't need that. See Security considerations below for what this does and doesn't protect against on its own.

## Security considerations

The plaintext credential no longer lives inside content that gets distributed everywhere. The old `.mobileconfig` VPN payload baked the shared secret into a file that every enrolled device received and stored, that Fleet stored in its own database, and that anyone with access to the device could recover through `profiles show` or the keychain. With assets, the declaration itself carries only a URL, and the secret exists in exactly one place you control. That also changes how rotation works: to rotate the secret, you change the JSON file in one place. You don't have to redistribute a new profile to every device the way the old model required.

The device is willing to present its MDM identity certificate as part of the request, per Apple's spec: "A request that uses MDM semantics, which includes the device-identity certificate, and any user authentication. This is equivalent to an MDM request made to the `CheckInURL` or `ServerURL`." Fleet always sends the request this way. What varies is whether your hosting server actually asks for and checks that certificate. A plain HTTPS server can just ignore it and respond to anyone, which is the simpler option if your security requirements don't call for more.

For a production deployment, verify that the requester is actually one of your enrolled devices before serving the credential. See [How to secure externally hosted DDM assets](https://fleetdm.com/guides/securing-externally-hosted-ddm-assets) for two ways to do that: verifying the `Mdm-Signature` header Apple attaches to the request, or requiring mutual TLS and validating the device's certificate against Fleet's CA at the handshake.

## Hosting the credential content

Create a JSON file matching the username/password credential schema:

```json
{
    "UserName": "ikev2-psk",
    "Password": "<your-shared-secret>"
}
```

Host it at an HTTPS URL with a trusted certificate. It doesn't need to be a dedicated server, a static file behind any reverse proxy holding a valid cert works. The endpoint has no authentication of its own from Fleet's side, so scope network access to it the way you would any endpoint serving a literal secret: internal-only, behind an access list, or mTLS.

## Upload the asset declaration

In Fleet: **Controls > OS settings > Custom settings > Assets tab > Add asset**.

```json
{
    "Type": "com.apple.asset.credential.userpassword",
    "Identifier": "<a-uuid-you-generate>",
    "ServerToken": "<a-uuid-you-generate>",
    "Payload": {
        "Reference": {
            "DataURL": "https://your-host/vpn-credential.json",
            "ContentType": "application/json"
        }
    }
}
```

Leave out the `Authentication` key. As covered above, Fleet always applies MDM authentication semantics to this fetch, so there's nothing to configure here, and including the key will cause the upload to fail validation.

## Upload the VPN declaration

In Fleet: **Controls > OS settings > Custom settings > Profiles tab > Add profile**.

```json
{
    "Type": "com.apple.configuration.network.vpn.ikev2",
    "Identifier": "<a-uuid-you-generate>",
    "ServerToken": "<a-uuid-you-generate>",
    "PayloadScope": "User",
    "Payload": {
        "VisibleName": "Your VPN name",
        "HostName": "your-vpn-server.example.com",
        "LocalIdentifier": "your-vpn-server.example.com",
        "RemoteIdentifier": "your-vpn-server.example.com",
        "Authentication": {
            "Method": "SharedSecret",
            "CredentialsAssetReference": "<the asset's Identifier from above>"
        }
    }
}
```

`Authentication.CredentialsAssetReference` points at the asset's `Identifier`, not its Fleet-assigned UUID. Upload the asset first. A VPN declaration that references an asset identifier Fleet doesn't already know about won't validate.

### Set `PayloadScope` to `User`

If you omit `PayloadScope`, it defaults to `System` (device channel). On macOS 27 Beta 2, that default produced a deployment that looked completely successful but silently didn't work. A device-scope IKEv2 declaration applied and loaded correctly, then a separate user-scope reconciliation pass ran seconds later, found no user-scope IKEv2 declarations, and removed the device-scope configuration anyway. `scutil --nc list` and System Settings both showed nothing afterward. No error surfaced anywhere, in Fleet or on the device. Fleet's declaration status stayed `verified` the whole time, since from the device's perspective it did apply the configuration correctly before something else removed it.

Setting `PayloadScope: "User"` explicitly put both applicator passes in the same scope and resolved it. We can't say with certainty whether this is intended behavior we were triggering by using the wrong scope or a beta-specific bug. Either way, set `PayloadScope` explicitly on VPN declarations rather than relying on the default.

## Verify the deployment

Fleet's declaration status (`Pending` → `Verifying` → `Verified`) confirms Fleet successfully delivered the declaration and the device acknowledged it. It does not confirm the VPN configuration is actually usable, per the scope issue above. To confirm it works:

- Check **System Settings > VPN** for the configuration to appear.
- Attempt an actual connection and confirm success both on the client and on your VPN server's logs (e.g., a successful `IKE_SA established` and `CHILD_SA established` in a strongSwan-based server's logs) rather than relying on the client UI alone.

## Related VPN declaration types

IKEv2 is one of four new VPN declaration types (`network.vpn.ikev2`, `network.vpn.ipsec`, `network.vpn.vpn-plugin`, `network.vpn.always-on`), each suited to a different protocol and authentication method. Always-On VPN is a singleton apply-mode declaration, so only one can be active on a device at a time.

## Further reading

- [`com.apple.configuration.network.vpn.ikev2` declaration reference](https://developer.apple.com/documentation/devicemanagement/networkvpnikev2)
- [Credential asset schema (`credential.userpassword.yaml`)](https://github.com/apple/device-management/blob/main/declarative/declarations/assets/credential.userpassword.yaml)
- [WWDC 2026: what IT admins need to know](https://fleetdm.com/guides/wwdc-2026-what-it-admins-need-to-know)

<meta name="articleTitle" value="Deploy VPN with declarative device management (DDM) in Fleet">
<meta name="authorFullName" value="Jake Stenger">
<meta name="authorGitHubUsername" value="jakestenger">
<meta name="publishedOn" value="2026-08-04">
<meta name="category" value="guides">
<meta name="description" value="A guide to deploying IKEv2 VPN declarations and credential assets in Fleet, including a scope pitfall that silently drops VPN configs.">

# MDM asset manager

The MDM Asset Manager is a tool designed to manage MDM assets in a Fleet database. The assets are
exported unencrypted and imported encrypted; hence, the
[key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) used for
encrypting the assets must be provided. 

## Usage

### Export

To export all MDM assets:

```
go run tools/mdm/assets/main.go export -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -dir=mdm_assets
```

You can also specify the asset you want:

```
go run tools/mdm/assets/main.go export -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -dir=mdm_assets -name=vpp_token
```

Supported flags are:

```
  -db-address string
    	Address used to connect to the MySQL instance (default "localhost:3306")
  -db-name string
    	Name of the database with the asset information in the MySQL instance (default "fleet")
  -db-password string
    	Password used to connect to the MySQL instance (default "insecure")
  -db-user string
    	Username used to connect to the MySQL instance (default "fleet")
  -dir string
    	Directory to put the exported assets
  -key string
    	Key used to encrypt the assets
  -name string
    	Name of the MDM asset to export
```

### Import

To import an MDM asset:

```
go run tools/mdm/assets/main.go import -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -name=vpp_token -value='{"foo": "bar"}'
```

Supported flags are:

```
  -db-address string
    	Address used to connect to the MySQL instance (default "localhost:3306")
  -db-name string
    	Name of the database with the asset information in the MySQL instance (default "fleet")
  -db-password string
    	Password used to connect to the MySQL instance (default "insecure")
  -db-user string
    	Username used to connect to the MySQL instance (default "fleet")
  -key string
    	Key used to encrypt the assets
  -name string
    	Name of the MDM asset to import
  -value string
    	Value to be set for the asset
```

### Rollover CA cert

Re-signs the Apple MDM CA certificate (`ca_cert`) in place, reusing the existing CA
private key so previously-issued SCEP client certificates remain valid. Use this when
the CA cert is approaching (or has passed) its expiry. **Stop all Fleet server
containers for the target deployment before running this**

```
go run tools/mdm/assets/main.go rollover-ca-cert -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -extend-years=5
```

The previous `ca_cert` row is soft-deleted (kept for audit) and the renewed cert is
inserted. The `ca_key` is untouched. A fresh serial is reserved from `identity_serials`
so the new CA cert can't collide with a client cert issued by the previous CA. After
restarting the servers, the "Fleet root certificate authority (CA)" profile redelivers
to all Apple-MDM-enrolled hosts on the next profile reconcile.

See [Rolling over the Apple MDM CA certificate](../../../docs/Contributing/guides/rollover-apple-mdm-ca-cert.md)
for the full procedure and expected side effects.

Supported flags are:

```
  -db-address string
    	Address used to connect to the MySQL instance (default "localhost:3306")
  -db-name string
    	Name of the database with the asset information in the MySQL instance (default "fleet")
  -db-password string
    	Password used to connect to the MySQL instance (default "insecure")
  -db-user string
    	Username used to connect to the MySQL instance (default "fleet")
  -extend-years int
    	Number of years to extend the Apple MDM CA certificate from now (default 5)
  -key string
    	Key used to encrypt the assets
```

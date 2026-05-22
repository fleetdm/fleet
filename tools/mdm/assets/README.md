# MDM asset manager

The MDM Asset Manager is a tool designed to manage MDM assets in a Fleet database. The assets are
exported unencrypted and imported encrypted; hence, the
[key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) used for
encrypting the assets must be provided.

## Configuration

The tool uses Fleet's standard MySQL configuration. Database options can be set with Fleet config
flags such as `--mysql_address`, `--mysql_username`, and `--mysql_tls_ca`; with `FLEET_MYSQL_*`
environment variables; or with a Fleet YAML config file passed via `--config`.

The encryption key can be provided with `--key`, or by setting `server.private_key` in the Fleet
config file.

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

Using TLS:

```
FLEET_MYSQL_TLS_CONFIG=skip-verify \
go run tools/mdm/assets/main.go export --key=... --dir=mdm_assets
```

Or with a custom CA:

```
FLEET_MYSQL_TLS_CA=/path/to/ca.pem \
FLEET_MYSQL_TLS_CERT=/path/to/client-cert.pem \
FLEET_MYSQL_TLS_KEY=/path/to/client-key.pem \
go run tools/mdm/assets/main.go export --key=... --dir=mdm_assets
```

Supported flags are:

```
  --config string
        Path to a Fleet configuration file
  --dir string
        Directory to put the exported assets
  --key string
        Key used to encrypt the assets
  --name string
        Name of the MDM asset to export
```

Fleet's standard `--mysql_*` flags are also available.

### Import

To import an MDM asset:

```
go run tools/mdm/assets/main.go import -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -name=vpp_token -value='{"foo": "bar"}'
```

Supported flags are:

```
  --config string
        Path to a Fleet configuration file
  --key string
        Key used to encrypt the assets
  --name string
        Name of the MDM asset to import
  --value string
        Value to be set for the asset
```

Fleet's standard `--mysql_*` flags are also available.

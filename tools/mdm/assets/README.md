# MDM asset manager

The MDM Asset Manager is a tool designed to manage MDM assets in a Fleet database. The assets are
exported unencrypted and imported encrypted; hence, the
[key](https://fleetdm.com/docs/configuration/fleet-server-configuration#server-private-key) used for
encrypting the assets must be provided.

## Environment Variables

All configuration can also be set via environment variables (prefix `ASSETS_DB_`). A `.env` file in
the current directory will also be loaded automatically if present.

| Environment Variable | Flag | Description | Default |
|---|---|---|---|
| `ASSETS_DB_USER` | `-db-user` | MySQL username | `fleet` |
| `ASSETS_DB_PASSWORD` | `-db-password` | MySQL password | `insecure` |
| `ASSETS_DB_ADDRESS` | `-db-address` | MySQL address | `localhost:3306` |
| `ASSETS_DB_NAME` | `-db-name` | MySQL database name | `fleet` |
| `ASSETS_DB_TLS_CONFIG` | `-tls-config` | TLS config name (e.g., `skip-verify`, `custom`) | `skip-verify` |
| `ASSETS_DB_TLS_CA` | `-tls-ca` | Path to CA certificate file | *(empty)* |
| `ASSETS_DB_TLS_CERT` | `-tls-cert` | Path to client certificate file | *(empty)* |
| `ASSETS_DB_TLS_KEY` | `-tls-key` | Path to client key file | *(empty)* |
| `ASSETS_DB_TLS_SERVER_NAME` | `-tls-server-name` | Server name for TLS verification | *(empty)* |

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
ASSETS_DB_TLS_CONFIG=skip-verify \
go run tools/mdm/assets/main.go export -key=... -dir=mdm_assets
```

Or with a custom CA:

```
ASSETS_DB_TLS_CA=/path/to/ca.pem \
ASSETS_DB_TLS_CERT=/path/to/client-cert.pem \
ASSETS_DB_TLS_KEY=/path/to/client-key.pem \
go run tools/mdm/assets/main.go export -key=... -dir=mdm_assets
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
  -tls-ca string
    	Path to the CA certificate file for MySQL TLS
  -tls-cert string
    	Path to the client certificate file for MySQL TLS
  -tls-config string
    	TLS configuration for MySQL connection (default "skip-verify")
  -tls-key string
    	Path to the client key file for MySQL TLS
  -tls-server-name string
    	Server name to use for MySQL TLS certificate verification
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
  -tls-ca string
    	Path to the CA certificate file for MySQL TLS
  -tls-cert string
    	Path to the client certificate file for MySQL TLS
  -tls-config string
    	TLS configuration for MySQL connection (default "skip-verify")
  -tls-key string
    	Path to the client key file for MySQL TLS
  -tls-server-name string
    	Server name to use for MySQL TLS certificate verification
  -value string
    	Value to be set for the asset
```

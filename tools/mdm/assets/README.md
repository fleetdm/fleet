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
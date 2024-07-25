## MDM asset extractor

The MDM Asset Extractor is a tool designed to extract all MDM assets from a
Fleet database. The assets are extracted unencrypted; hence, the key used for
encrypting the assets must be provided.

### Usage

During development, a typical usage example would be:

```
go run tools/mdm/assets/main.go -key=E6Ow1t2dbKARxEF6O9GFI3DDQRMROhI8 -dir=mdm_assets
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
```

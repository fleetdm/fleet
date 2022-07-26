## installerstore 

Small utility to upload installers to a file storage (AWS S3, MinIO, etc.)
using the keys where the Fleet server expects to find them.

```
NAME:
   installerstore - Utility to upload pre-built installers to a file storage (AWS S3, MinIO, etc.)

USAGE:
   installerstore --enroll-secret=xyz --bucket=installers ~/path/to/file.pkg

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --fleet-desktop value                                     Whether or not the installer includes Fleet Desktop [$INSTALLER_FLEET_DESKTOP]
   --enroll-secret value                                     Enroll secret associated with the installer [$INSTALLER_ENROLL_SECRET]
   --bucket value                                            Bucket where to store installers [$INSTALLER_BUCKET]
   --prefix value                                            Prefix under which installers are stored [$INSTALLER_PREFIX]
   --region value                                            AWS Region (if blank region is derived) [$INSTALLER_REGION]
   --endpoint-url value                                      AWS Service Endpoint to use (leave blank for default service endpoints) [$INSTALLER_ENDPOINT_URL]
   --access-key-id value                                     Access Key ID for AWS authentication [$INSTALLER_ACCESS_KEY_ID]
   --secret-access-key value                                 Secret Access Key for AWS authentication [$INSTALLER_SECRET_ACCESS_KEY]
   --sts-assume-role-arn value                               ARN of role to assume for AWS [$INSTALLER_STS_ASSUME_ROLE_ARN]
   --disable-ssl                                             Disable SSL (typically for local testing) (default: false) [$INSTALLER_DISABLE_SSL]
   --force-s3-path-style http://s3.amazonaws.com/BUCKET/KEY  Set this to true to force path-style addressing, i.e., http://s3.amazonaws.com/BUCKET/KEY (default: false) [$INSTALLER_FORCE_S3_PATH_STYLE]
   --create-bucket                                           Set this to true to create the bucket if it doesn't exist. Only recommended for local testing. (default: false) [$INSTALLER_CREATE_BUCKET]
   --help, -h                                                show help (default: false)
```

### Example

To upload a file for testing to your local MinIO server, you can run this
command from the root of the repo (be sure to replace the `--enroll-secret` 
string with the value you wish to test and set the `--fleet-desktop` boolean 
to your desired value):

```
go run tools/installerstore/main.go \
  --enroll-secret=xyz \
  --bucket=installers-dev \
  --region=minio \
  --prefix=dev-prefix \
  --endpoint-url=localhost:9000 \
  --access-key-id=minio \
  --secret-access-key=minio123! \
  --disable-ssl=true \
  --force-s3-path-style=true \
  --create-bucket=true \
  --fleet-desktop=true \
  fleet-osquery.pkg
```

Tip: MinIO provides an UI you can use to explore your local buckets. If you're
running the Fleet server in development, it shoud be available at
http://localhost:9001.

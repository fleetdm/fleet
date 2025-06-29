# Testing large quantities of software packages in Fleet

1. download the following folder of packages:
[Google Drive Packages](https://drive.google.com/drive/folders/1aDmQGBSNCmKatyQbQVKlLi9NboTEgoxM?usp=drive_link)

2. Run the following script to upload a folder of packages via the Fleet API:

``` bash
bash ./tools/software/packages/upload-packages.sh -u $FLEET_URL -t $TEAM_ID -k $API_KEY -f $FOLDER_PATH_CONTAINING_PACKAGES
```

Notes:

- uploading to a `localhost` Fleet server is heavily encouraged as uploading a large amount of packages is time consuming and can be subject to timeouts using tools like `ngrok`
- `upload-packages.sh` creates non-functional install and uninstall scripts for `exe` files

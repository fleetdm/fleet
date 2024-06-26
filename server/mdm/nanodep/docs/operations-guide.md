# NanoDEP Operations Guide

This is a brief overview of the various tools and utilities for working with NanoDEP.

## DEP names

NanoDEP supports configuring multiple DEP "MDM servers." These different DEP "MDM servers" are referenced by an arbitrary name string that you specify. This string is used to both configure the DEP connection (like authentication) as well to reference these configuration for actually talking to the Apple DEP API endpoints.

Note that because the name string is used pervasively in URL API paths you probably want to avoid names that include things like forward-slashes "/", spaces, or anything else really that might have trouble in URLs.

## depserver

The `depserver` serves two main purposes:

1. Setup & configuration of the DEP name(s) — that is, the locally-named instances that correspond to the DEP "MDM servers" in the Apple Business Manager (ABM), Apple School Manager (ASM), or Business Essentials (BE) portal. Configuration includes uploading the DEP authentication tokens, configuring the assigner, etc. See the "API endpoints" section below for more.
1. Accessing the actual DEP APIs using a transparently-authenticating reverse proxy. After you've configured the authentication tokens using the above APIs `depserver` provides a reverse proxy to talk to the Apple DEP endpoints where you don't have to worry about session management or token authentication: this's taken care of for you. All you need to do is use a special URL path and normal API (HTTP Basic) authentication and you can talk to the DEP APIs unfiltered. See the "Reverse proxy" section below for more.

### Switches

Command line switches for the `depserver` tool.

#### -api string

* API key for API endpoints

Required. API authentication in NanoDEP is simply HTTP Basic authentication using "depserver" as the username and the API key (from this switch) as the password.

#### -debug

* log debug messages

Enable additional debug logging.

#### -listen string

* HTTP listen address (default ":9001")

Specifies the listen address (interface and port number) for the server to listen on.

#### -storage & -storage-dsn

The `-storage` and `-storage-dsn` flags together configure the storage backend. `-storage` specifies the name of backend type while `-storage-dsn` specifies the backend data source name (e.g. the connection string). If no `-storage` backend is specified then `file` is used as a default.

##### file storage backend

* `-storage file`

Configure the `file` storage backend. This backend manages DEP authentication and configuration data within plain filesystem files and directories. It has zero dependencies and should run out of the box. The `-storage-dsn` flag specifies the filesystem directory for the database. If no `storage-dsn` is specified then `db` is used as a default.

*Example:* `-storage file -storage-dsn /path/to/my/db`

##### mysql storage backend

* `-storage mysql`

Configures the MySQL storage backend. The `-dsn` flag should be in the [format the SQL driver expects](https://github.com/go-sql-driver/mysql#dsn-data-source-name).
Be sure to create the storage tables with the [schema.sql](../storage/mysql/schema.sql) file. MySQL 8.0.19 or later is required.

*Example:* `-storage mysql -dsn nanodep:nanodep/mydepdb`

#### -version

* print version

Print version and exit.

### API endpoints

API endpoints for getting and setting the configuration of DEP names. Note that you don't need to use these APIs directly — NanoDEP provides a set of tools and scripts for working with some of these endpoints — see the "Tools and scripts" section, below. Most of the endpoints require specifying the "DEP name" (see above) in the `{name}` part of the URL (without the curly braces, of course).

A brief overview of the endpoints is provided here. For detailed API semantics please see the [OpenAPI documentation for NanoDEP](https://www.jessepeterson.space/swagger/nanodep.html). The OpenAPI source YAML is a part of this project.

#### Version

* Endpoint: `GET /version`

Returns a JSON response with the version of the running NanoDEP server.

#### Token PKI

* Endpoint: `GET, PUT /v1/tokenpki/{name}`

The `/v1/tokenpki/{name}` endpoints deal with the public key exchange using the Apple ABM/ASM/BE portal for acquiring the authentication tokens for talking to the DEP API. For example usage please see the `./tools/cfg-get-cert.sh` and `./tools/cfg-decrypt-tokens.sh` scripts. These scripts are talked about under section "Tools and scripts" below.

#### Tokens

* Endpoint: `GET, PUT /v1/tokens/{name}`

The `/v1/tokens/{name} ` endpoints deal with the raw DEP OAuth tokens in JSON form. I.e. after the PKI exchange you can query for the actual DEP OAuth tokens if you like. This also allows configuring the OAuth1 tokens for a DEP name if you already have the tokens in JSON format. I.e. if you used the `deptokens` tool or you're using the DEP simulator `depsim`.

#### Assigner

* Endpoint: `GET, PUT /v1/assigner/{name}`

The `/v1/assigner/{name}` endpoints deal with storing and retrieving the assigner profile UUID. This is used for the assigner tool `depsyncer` (see below for documentation on that tool). For example usage please see the `./tools/cfg-set-assigner.sh` script. This script is talked about under section "Tools and scripts" below.

#### Config

* Endpoint: `GET, PUT /v1/config/{name}`

The `/v1/config/{name}` endpoints deal with storing and retrieving configuration for a given DEP name. At this time the only configuration available is the base URL of the DEP name. This is really only useful when talking to the DEP simulator `depsim` or perhaps directing DEP server requests through another reverse proxy.

### Reverse proxy

In addition to individually handling some of various Apple DEP API endpoints in its `godep` library NanoDEP provides a transparently-authenticating HTTP reverse proxy to the Apple DEP servers. This allows us to simply provide `depserver` with the Apple DEP endpoint, the NanoDEP "DEP name" and the API key, and we can talk to any of the Apple DEP endpoint APIs (including the Roster, Class, and People Management). `depserver` will authenticate to the Apple DEP server and keep track of session management transparently behind the scenes. To be clear: this means you do not have to call to the `/session` endpoint to authenticate nor to manage and update the session tokens with each request. NanoDEP does this for you.

The proxy URL is accessible as: `/proxy/{name}/endpoint` where `/endpoint` is the Apple DEP API endpoint you want to access. The proxy will automatically translate this URL to ``https://mdmenrollment.apple.com/endpoint` and use `{name}` for retrieving the DEP authentication tokens. Note that in some cases, for some endpoints, various HTTP headers are added or removed:

* For any proxy request the API authentication header is removed before passing to the underlying DEP server.
* If not provided in the incoming HTTP request the DEP header `X-Server-Protocol-Version` is set to a default (currently "3").
* For the `/session` endpoint we use a default `Content-Type`. However because NanoDEP handles authentication for you, you shouldn't have to worry about this (or even need to call to the `/session` endpoint).

Note that for simple cases you don't need to use this proxy directly — NanoDEP provides a set of tools and scripts for working with some of the DEP endpoints — see the "Tools and scripts" section, below.

#### Example usage

You can see this example alternatively as `./tools/dep-account-detail.sh` under the "Tools and scripts" section, below, but we'll duplicate it for illustrative purposes:

```bash
% curl -v -u depserver:supersecret 'http://[::1]:9001/proxy/mdmserver1/account'
*   Trying ::1...
* TCP_NODELAY set
* Connected to ::1 (::1) port 9001 (#0)
* Server auth using Basic with user 'depserver'
> GET /proxy/mdmserver1/account HTTP/1.1
> Host: [::1]:9001
> Authorization: Basic ZGVwc2VydmVyOnN1cGVyc2VjcmV0
> User-Agent: curl/7.64.1
> Accept: */*
> 
< HTTP/1.1 200 OK
< Content-Length: 321
< Content-Type: application/json;charset=UTF8
< Date: 2022-07-04T15:06:54-07:00
< X-Adm-Auth-Session: 982B2965F9C9D9672EDA4BAF7902755657480328585B9871D1022898C04A3419BA24771734DB2031FF122E0E789EE347AF89E80EBDA521A429C2F90FE7F9031E
< 
* Connection #0 to host ::1 left intact
{
  "server_name": "Example Server",
  "server_uuid": "677cab70-fe18-11e2-b778-0800200c9a66",
  "facilitator_id": "facilitator@example.com",
  "org_phone": "111-222-3333",
  "org_name": "Example Inc",
  "org_email": "orgadmin@example.com",
  "org_address": "123 Main St. Anytown, USA",
  "admin_id": "admin@example.com"
}
* Closing connection 0
```

This request was 'translated' from `GET /proxy/mdmserver1/account` to `GET /account` at the `https://mdmenrollment.apple.com` URL and authenticated using the `mdmserver1` DEP name (assuming it was already configured, of course). You can also see the returned `X-Adm-Auth-Session` header which contains the response session token (which you can ignore because NanoDEP handles dealing with this header under the hood).

## Tools and scripts

The NanoDEP project includes some tools and scripts that use the above APIs in `depserver` for performing some typical DEP device management tasks. These are basically just shell scripts that utilize `curl` and `jq` to drive the `depserver` API and/or Apple DEP API endpoints (and so, obviously, those tools are requirements for the scripts to work). These tools and scripts also have their own documentation under the `./tools` directory of the project as noted below.

Generally, the scripts are split into two types indicated by the script prefix:

* Scripts starting with `cfg-` configure `depserver` and its properties.
* Scripts starting with `dep-` use the Reverse Proxy described above to perform operations with the Apple DEP server.

### Scripts

These scripts require setting up a few environment variables before use. Please see the [tools](../tools) for more documentation. But generally you'll need to set these environment variables for the scripts to work:

```bash
# the URL of the running depserver
export BASE_URL='http://[::1]:9001'
# should match the -api switch of the depserver
export APIKEY=supersecret
# the DEP name (instance) you want to use
export DEP_NAME=mdmserver1
```

The [Quickstart Guide](quickstart.md) also documents some usage of these scripts, too.

#### cfg-get-cert.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script generates and retrieves the public key certificate for use when downloading the DEP authentication tokens from the ABM/ASM/BE portal. The `curl` call will dump the PEM-encoded certificate to stdout so you'll likely want to redirect it somewhere useful so it can be uploaded to the portal.

##### Example usage

```bash
$ ./tools/cfg-get-cert.sh > $DEP_NAME.pem
  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
                                 Dload  Upload   Total   Spent    Left  Speed
100  1001  100  1001    0     0   4509      0 --:--:-- --:--:-- --:--:--  4509
$ head -2 $DEP_NAME.pem
-----BEGIN CERTIFICATE-----
MIICtTCCAZ2gAwIBAgIBATANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDEwlkZXBz
```

#### cfg-decrypt-tokens.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script uploads the encrypted tokens that were downloaded from the ABM/ASM/BE portal to `depserver` where it is decrypted and the resulting OAuth tokens stored with the MDM instance.

**The first argument is required** and specifies the path to the token file downloaded from the Apple portal.

##### Example usage

```bash
$ ./cfg-decrypt-tokens.sh ~/Downloads/mdmserver1_Token_2022-07-01T22-18-53Z_smime.p7m
{"consumer_key":"CK_9af2f8218b150c351ad802c6f3d66abe","consumer_secret":"CS_9af2f8218b150c351ad802c6f3d66abe","access_token":"AT_9af2f8218b150c351ad802c6f3d66abe","access_secret":"AS_9af2f8218b150c351ad802c6f3d66abe","access_token_expiry":"2023-07-01T22:18:53Z"}
```

#### dep-account-detail.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script queries the DEP API [Get Account Detail](https://developer.apple.com/documentation/devicemanagement/get_account_detail) endpoint and returns the data.

##### Example usage

```bash
$ ./dep-account-detail.sh
{
  "server_name": "Example Server",
  "server_uuid": "677cab70-fe18-11e2-b778-0800200c9a66",
  "facilitator_id": "facilitator@example.com",
  "org_phone": "111-222-3333",
  "org_name": "Example Inc",
  "org_email": "orgadmin@example.com",
  "org_address": "123 Main St. Anytown, USA",
  "admin_id": "admin@example.com"
}
```

#### dep-define-profile.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script uploads a [DEP profile](https://developer.apple.com/documentation/devicemanagement/profile) in JSON form to the Apple DEP API [Define A Profile](https://developer.apple.com/documentation/devicemanagement/define_a_profile) endpoint. Some important notes:

* **The first argument is required** and specifies the path to a DEP profile JSON file. We provide a sample DEP profile in the [docs](../docs) of the NanoMDM project to get you started.
* *You will need to (possibly heavily) modify this example* including MDM server URL, adding or removing optional parameters, devices serial numbers to assign to, etc. See the Apple [DEP profile](https://developer.apple.com/documentation/devicemanagement/profile) documentation and test extensively. Note some properties in the profile are mutually exclusive and the DEP service doesn't always given good feedback. Trial and error is sometimes need to get your first DEP profile uploaded successfully.
* You can directly include `devices` key in the JSON here to assign this profile *during this operation* to those devices. This means you can skip a separate device assign step which would be required.
* Once uploaded to Apple the profile will have a UUID associated with it. This identifies this exact uploaded profile to Apple for future reference. You may want to note this profile UUID if, for example, you want to use it to automatically assign devices with the `depsyncer` tool.

##### Example usage

```bash
$ ./dep-define-profile.sh ../docs/dep-profile.example.json
{
  "profile_uuid": "43277A13FBCA0CFC",
  "devices": {
    "07AAD449616F566C12": "SUCCESS"
  }
}
```

#### dep-device-details.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script queries the Apple DEP API [Get Device Details](https://developer.apple.com/documentation/devicemanagement/get_device_details) endpoint for a given serial number.

**The first argument is required** and specifies the serial number of the device you want to query.

Note that the API itself supports querying multiple devices at a time if you're able to assemble the appropriate JSON. This script only supports one serial number, however.

##### Example usage

```bash
$ ./dep-device-details.sh 07AAD449616F566C12
{
  "devices": {
    "07AAD449616F566C12": {
      "serial_number": "07AAD449616F566C12",
      "profile_uuid": "43277A13FBCA0CFC",
...
```

#### dep-get-profile.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script queries the Apple DEP API [Get a Profile](https://developer.apple.com/documentation/devicemanagement/get_a_profile) endpoint for a given DEP Profile UUID.

**The first argument is required** and specifies the UUID of the profile that was previously defined via the API.

#####

```bash
$ ./dep-get-profile.sh 43277A13FBCA0CFC
{
  "profile_uuid": "43277A13FBCA0CFC",
...
```

#### cfg-set-assigner.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script saves the 'assigner' profile UUID in the `depserver` storage backend. This is the profile UUID that the automatic DEP profile assigner in the `depsyncer` tool uses to assign serial numbers to as it syncs new devices. By itself this command doesn't actually assign profiles to anything — it only *configures* the assigner profile UUID. The endpoint responds with the profile UUID in JSON. See the `depsyncer` tool documentation for more information.

**The first argument is required** and specifies the UUID of the profile that `depsyncer` will use to automatically assign serial numbers to.

##### Example usage

```bash
$ ./cfg-set-assigner.sh 43277A13FBCA0CFC
{"profile_uuid":"43277A13FBCA0CFC"}
```

#### dep-remove-profile.sh

For the DEP "MDM server" in the environment variable $DEP_NAME (see above) this script calls to the Apple DEP API [Remove a Profile](https://developer.apple.com/documentation/devicemanagement/remove_a_profile-c2c) endpoint to remove a serial number from being assigned to a DEP profile UUID. Note this is **NOT** the [disown](https://developer.apple.com/documentation/devicemanagement/disown_devices) endpoint and profiles can be re-assigned at any time after using this script.

**The first argument is required** and specifies the serial number of the device to remove DEP profile assignment from.

Note that the API itself supports un-assigning multiple devices at a time if you're able to assemble the appropriate JSON. This script only supports one serial number, however.

##### Example usage

```bash
$ ./dep-remove-profile.sh 07AAD449616F566C12
{
  "devices": {
    "07AAD449616F566C12": "SUCCESS"
  }
}
```

### Troubleshooting

Sometimes something goes wrong with the API or the scripts. Sometimes the API will tell you exactly the problem you have and how you can fix the input. Other times you may need to inspect the HTTP request details. To do that you can turn on `curl` verbose output by setting the `CURL_OPTS` environment variable which all the scripts utilize:

```bash
export CURL_OPTS=-v
```

And then run the script again. This should give detailed HTTP response data including headers, etc.

## depsyncer

`depsyncer` is a stand-alone tool for syncing devices from the Apple DEP service. It operates by continuously syncing the list of the devices from the Apple DEP "MDM server" configurations. `depsyncer` can optionally assign DEP profiles to newly added devices as it syncs devices. `depsyncer` can also optionally send a webhook HTTP call to a webserver with the synced device information.

Note that `depsyncer` does not itself save any of the synced device information. The synced devices are either assigned a DEP profile or sent off to a webhook URL — ostensibly for any custom processing or saving to databases or such.

### Assignment

`depsyncer` can optionally assign DEP profiles to newly added devices as it syncs them. For each set of synced devices the auto-assigner will read the storage backend's configured assigner profile UUID for the given DEP name and attempt to assign the devices to it as they are synced.

You can set the assigner profile UUID using either the `./tools/cfg-set-assigner.sh` script (which talks to `depserver`) or using the `depserver` API endpoint `/v1/assigner/{name}` directly. See above for documentation on either of these options. The assigner can be set or changed at any time — even if `depsyncer` has already started: it reads the profile UUID every sync cycle. Note also that the assigner profile UUID applies only to the specific associated DEP name.

### Usage

At minimum you must specify at least one DEP name to start syncing devices from:

```bash
$ ./depsyncer-darwin-amd64 -h
Usage: ./depsyncer-darwin-amd64 [flags] <DEPname1> [DEPname2 [...]]
Flags:
...
```

Other than the switches (flags) documented below you just specify the DEP names that you'd like to sync (and assign) devices from. Multiple syncers will start up for each DEP name provided.

Examples in the "Example usage" section are below.

## Signals

When run in "continuous" mode (the default) `depsyncer` waits for a duration between syncing devices. During this wait period you can request an explicit sync by sending the `depsyncer` process a Signal hangup (SIGHUP). For example if your system has the `killall` command and your `depsyncer` binary is called `depsyncer-darwin-amd64` you could:

```bash
$ killall -SIGHUP depsyncer-darwin-amd64
```

You should then see in the running `depsyncer` process:

```bash
2022/07/06 15:40:14 level=debug component=syncer name=depsim msg=device sync: explicit sync requested
```

Whereby the next sync should be immediately started. Naturally signal handling is OS dependent and so this feature will not work on Microsoft Windows. `depsyncer` also tries to handle the Interrupt and Terminate (SIGTERM) signals to try to cleanly stop the syncer(s) and shutdown the process.

### Switches (flags)

#### -debug

* log debug messages

Enable additional debug logging.

#### -debug-assigner

* additional debug logging of the device assigner

Enable extra debug logging for the device assigner component specifically.

#### -duration uint

* duration in seconds between DEP syncs (0 for single sync) (default 1800)

If `-duration` is 0 then `depsyncer` only performs a single sync (and assign) cycle for each provided DEP name and then exits ("sync once" mode). Because `depsyncer` saves the cursor provided by the Apple DEP API it knows how to pick up where it left off if it is run again.

If `-duration` is greater than 0 (the default) then `depsyncer` will never exit and continually run barring any errors ("continuous" mode). For each provided DEP name it will start an initial sync (and assign) cycle, then wait until the given duration has passed and start another sync cycle picking up where it left off.

In the "sync once" mode (duration of 0) `depsyncer` could be run from, say, a cron job or other task schedular. Note the sync is technically more efficient when run in "continuous" mode, API-wise, as it skips the "fetch" step once it has been completed once during each startup. Of course this could be offset by the lower resource utilization or greater flexibility of using the "sync once" mode.

#### -limit int

* limit fetch and sync calls to this many devices (0 for server default)

The limit flag specifies how many devices to fetch at a time from the Apple DEP API. [Apple's documentation](https://developer.apple.com/documentation/devicemanagement/syncdevicerequest) says there is a server-side default of 100 an upper limit of 1000.

#### -storage & -storage-dsn

See the "-storage & -storage-dsn" section, above, for `depserver`. The syntax and capabilities are the same.

#### -version

* print version

Print version and exit.

#### -webhook-url string

* URL to send requests to

For each synced set of devices `depsyncer` supports sending the sync result to a webhook URL. This switch turns on the webhook and specifies the URL. This is somewhat compatible with the webhook support in NanoMDM as well as the [MicroMDM webhook](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md).

##### Webhook data

The data is sent as an HTTP POST method with JSON data as the raw body. The JSON structure is similar to other open source webhook styles with a few differences:

* The top-level "topic" key will be a string of either `dep.SyncDevices` or `dep.FetchDevices` depending on the type of DEP API request used.
* The top-level "device_response_event" object will contain specific detail about this sync.
  * The key "dep_name" corresponds to the NanoDEP DEP name from which devices were synced.
  * The key "device_response" will be an object that corresponds to the Apple DEP API [FetchDeviceResponse](https://developer.apple.com/documentation/devicemanagement/fetchdeviceresponse) structure and includes the list of [Device](https://developer.apple.com/documentation/devicemanagement/device)(s) that were synced, if any.

With this information you could, for example, take device-specific actions by calling back into the `depserver` DEP APIs. For example to assign different DEP profiles depending on groups of serial numbers that you maintain or *not* assigning some serial numbers. It's all up to you with the DEP sync data provided.

##### Example data

Example JSON webhook body data:

```json
{
  "topic": "dep.SyncDevices",
  "event_id": "",
  "created_at": "2022-07-08T01:17:52.778653-07:00",
  "device_response_event": {
    "dep_name": "mdmserver1",
    "device_response": {
      "cursor": "MTY1NzI2ODE5Ny0x",
      "fetched_until": "0001-01-01T00:00:00Z",
      "more_to_follow": false,
      "devices": [
        {
          "serial_number": "07AAD449616F566C12",
          "op_type": "added",
          ...
        }
      ]
    }
  }
}
```

### Example usage

For the simplest invocation you can start `depsyncer` with is just a DEP name:

```bash
$ ./depsyncer-darwin-amd64 mdmserver1
2022/07/06 22:27:18 level=info component=syncer name=mdmserver1 msg=device sync phase=fetch more=false cursor=MTY1NzE0NzA5My0x devices=1 fetched_until=2022-07-06 15:38:13 -0700 PDT
2022/07/06 22:27:19 level=info component=syncer name=mdmserver1 msg=device sync phase=sync more=false cursor=MTY1NzE3MTU3Mi0w devices=0
```

If we wanted to specify a more complex startup we might:

* Set the device limit to 200 (over the Apple server default of 100)
* Double the default sync duration to an hour
* Ask for debug logging output
* Specify two DEP names to sync devices from.

```bash
$ ./depsyncer-darwin-amd64 -debug -limit 200 -duration 3600 mdmserver1 mdmserver2             
2022/07/06 23:32:06 level=debug component=syncer name=mdmserver2 msg=starting timer duration=1h0m0s
2022/07/06 23:32:06 level=debug component=syncer name=mdmserver1 msg=starting timer duration=1h0m0s
2022/07/06 23:32:06 level=debug component=syncer name=mdmserver2 msg=cursor returned all devices previously phase=fetch cursor=MTY1NzE3MjcxNy0x
2022/07/06 23:32:06 level=debug component=syncer name=mdmserver1 msg=cursor returned all devices previously phase=fetch cursor=MTY1NzE3NTI2My0w
2022/07/06 23:32:06 level=info component=syncer name=mdmserver1 msg=device sync phase=sync more=false cursor=MTY1NzE3NTQ1OS0w devices=0
2022/07/06 23:32:06 level=info component=syncer name=mdmserver2 msg=device sync phase=sync more=false cursor=MTY1NzE3MTU3Mi0w devices=2 fetched_until=2022-06-27 22:37:58 +0000 UTC op_type_added=2
```

Here we can see that both syncers started and the syncer for DEP name "mdmserver2" had two added devices.

To perform a single sync which then exits we can use the `-duration 0` switch (notice no timer being started):

```bash
$ ./depsyncer-darwin-amd64 -debug -duration 0 depsim                              
2022/07/07 00:05:45 level=debug component=syncer name=depsim msg=cursor returned all devices previously phase=fetch cursor=MTY1NzE3NTczOC0w
2022/07/07 00:05:45 level=info component=syncer name=depsim msg=device sync phase=sync more=false cursor=MTY1NzE3NzQ3OC0w devices=0
```

### Troubleshooting

The `depsyncer` tool has two debug switches: one for general debug logging (`-debug`) and another debug logging specifically for the assigner (`-debug-assigner`). Turning these options on may give extra detail into the which devices the syncer is seeing and which it is considering for assignment.

If you're moving devices from an existing MDM server in ABM/ASM/BE to your NanoDEP server then you may encounter a situation where, after moving the device over, you have `op_type_modified=X` but *not* `op_type_added=Y` device log lines (the former are required fot the assigner to work). This appears to be an oddity with the ABM/ASM/BE portal that [the MicroMDM project has documented](https://github.com/micromdm/micromdm/wiki/DEP-auto-assignment#reassignment-oddities) with a (kludgy) workaround.

## deptokens

The `deptokens` tool is an *optional* small stand-alone utility for decrypting the DEP OAuth tokens from the ABM/ASM/BE portal. It operates in one of two modes depending on if the `-token` switch is provided.

In "keypair generation" mode (that is, without specifying the `-token` switch) it will generate an RSA private key and certificate and save them both to disk (the private key optionally encrypted with the `-password` switch). The path to the certificate and key are provided in the `-cert` and `-key` switches, respectively.

In "decrypt and decode tokens" mode (that is, by specifying the path to the downloaded tokens file with the `-token` switch) it will attempt to use the certificate and key on disk (specified by `-cert` and `-key` switches, respectively, with an optional password for an encrypted private key specified with `-password`) to decrypt the tokens and display them. They can then be stored in `depserver` by using the "raw" token API (documented above).

**Note: `deptokens` is not required to use NanoDEP: `depserver` contains this functionality built-in using the tools/scripts (or via the API) directly. See above documentation.**

### Switches

Command line switches for the `deptokens` tool.

#### -cert string

* path to certificate (default "cert.pem")

The file path to read or save the X.509 certificate that contains the public key that the DEP OAuth tokens will be encrypted to.

#### -f

* force overwriting the keypair

By default `deptokens` tries not to overwrite the certificate or private key if a file exists at those paths. If this switch is provided it will happily overwrite them.

#### -key string

* path to key (default "cert.key")

The file path to read or save the RSA private key that corresponds to the public key that the OAuth tokens will be encrypted to.

#### -password string

* password to encrypt/decrypt private key with

A password to encrypt or decrypt RSA private key on disk with. Note this is password is just to protect the private key itself and does not play a role in the token PKI exchange with Apple.

#### -token string

* path to tokens

The file path to the ".p7m" file that Apple generates when retrieving the encrypted OAuth tokens from the ABM/ASM/BE portal.

If this switch is missing (the default) `deptokens` operates in "keypair generation" mode. If this switch is provided `deptokens` operates in "decrypt and decode tokens" mode. This follows the two-step upload-certificate then download-token process required for retrieving the DEP OAuth tokens.

#### -version

* print version

Print version and exit.

### Example usage

#### Keypair generation

```bash
$ ./deptokens-darwin-amd64 -password supersecret
wrote cert.pem, cert.key
```

Here `deptokens` wrote a PEM encoded certificate to `cert.pem` and a password encrypted private key to `cert.key`. We can upload `cert.pem` to Apple's ASM/ABM/BE portal as usual (see above or the quick start guide).

#### Decrypt and decode tokens

```bash
$ ./deptokens-darwin-amd64 -password supersecret -token /Users/negacctbal/Downloads/mdmserver1_Token_2022-07-06T06-03-00Z_smime.p7m
{"consumer_key":"CK_9af2f8218b150c351ad802c6f3d66abe","consumer_secret":"CS_9af2f8218b150c351ad802c6f3d66abe","access_token":"AT_9af2f8218b150c351ad802c6f3d66abe","access_secret":"AS_9af2f8218b150c351ad802c6f3d66abe","access_token_expiry":"2023-07-01T22:18:53Z"}
```

Here `deptokens` has read the default paths for the certificate and private key (`cert.pem` and `cert.key` respectively), decrypted the private key using the `-password` switch and using this private key decrypted the token file provided using the `-token` switch. It dumped the decrypted OAuth tokens JSON to stdout.

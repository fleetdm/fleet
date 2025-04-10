# NanoMDM Operations Guide

This is a brief overview of the various command-line switches and HTTP endpoints (including APIs) available to NanoMDM.

## Enrollment IDs

First, an aside on NanoMDM enrollment IDs:

Generally speaking in Apple MDM there are two types of MDM "channels" — the device channel and user channel. The device channel has different styles of enrollment. For example a traditional MDM device enrollment which would use the `UDID` field or a User Enrollment (for BYOD) which use the `EnrollmentID` field. Then, for the user channel there's an optional `UserID` field. This field changes context if it's a Shared iPad enrollment.

NanoMDM tries to reduce this complexity by collapsing these various identifiers into a single "enrollment ID" which is a single string that identifies unique enrollments. This same enrollment ID is used for targeting commands and pushes to devices. In the code we ["resolve"](https://github.com/micromdm/nanomdm/blob/5a0a160c8d89259bdd5feca346c0a9c4a93f95cc/mdm/type.go#L69) the various identifiers to their channel- and enrollment-types and then the core NanoMDM service ["normalizes"](https://github.com/micromdm/nanomdm/blob/5a0a160c8d89259bdd5feca346c0a9c4a93f95cc/service/nanomdm/service.go#L34-L45) the resolved IDs into enrollment IDs. The enrollment IDs look a bit like this:

| Type    | Platform | ID Normalized | ID Example |
| ------- | -------- | ------------- | ------- |
| Device  | macOS    | `UUID`        | `470E005B-17C1-4537-BBB3-0EBC340D432A` |
| User    | macOS    | `UUID:UUID` | `470E005B-17C1-4537-BBB3-0EBC340D432A:F151140B-3988-45A9-9471-E96B49F27D93` |
| Device  | iOS      | `UUID`        | `8b3b8ba3783e9ade1dae4fbb944ab3afc0ce5b69` |
| User Enrollment (Device) | iOS | `UUID` | `b318edb72b556059a013368e3150050c5f74a2c6` |
| Shared iPad | iOS  | `UUID:ShortName` | `68656c6c6f776f726c6468656c6c6f776f726c64:appleid@example.com` |

## Switches

###  -api string

* API key for API endpoints

API authorization in NanoMDM is simply HTTP Basic authentication using "nanomdm" as the username and the API key as the password. Omitting this switch turns off all API endpoints — NanoMDM in this mode will essentially just be for handling MDM client requests. It is not compatible with also specifying `-disable-mdm`.

### -ca string

* Path to CA cert for verification

NanoMDM validates that the device identity certificate is issued from specific CAs. This switch is the path to a file of PEM-encoded CAs to validate against.

### -cert-header string

* HTTP header containing URL-escaped TLS client certificate

By default NanoMDM tries to extract the device identity certificate from the HTTP request by decoding the "Mdm-Signature" header. See ["Pass an Identity Certificate Through a Proxy" section of this documentation for details](https://developer.apple.com/documentation/devicemanagement/implementing_device_management/managing_certificates_for_mdm_servers_and_devices). This corresponds to the `SignMessage` key being set to true in the enrollment profile.

With the `-cert-header` switch you can specify the name of an HTTP header that is passed to NanoMDM to read the client identity certificate. This is ostensibly to support Nginx' [$ssl_client_escaped_cert](http://nginx.org/en/docs/http/ngx_http_ssl_module.html) in a [proxy_set_header](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_set_header) directive. Though any reverse proxy setting a similar header could be used, of course. The `SignMessage` key in the enrollment profile should be set appropriately.

### -checkin

* enable separate HTTP endpoint for MDM check-ins

By default NanoMDM uses a single HTTP endpoint (`/mdm` — see below) for both commands and results *and* for check-ins. If this option is specified then `/mdm` will only be for commands and results and `/checkin` will only be for MDM check-ins.

### -debug

* log debug messages

Enable additional debug logging.

### -storage, -storage-dsn, & -storage-options

The `-storage`, `-storage-dsn`, & `-storage-options` flags together configure the storage backend(s). `-storage` specifies the name of the backend while `-storage-dsn` specifies the backend data source name (e.g. the connection string). The optional `-storage-options` flag specifies options for the backend if it supports them. If no storage flags are supplied then it is as if you specified `-storage file -storage-dsn db` meaning we use the `file` storage backend with `db` as its DSN.

_Note:_ NanoMDM versions v0.5.0 and below used the `-dsn` flag while later versions switched to the `-storage-dsn` flag.

#### file storage backend

* `-storage file`

Configures the `file` storage backend. This manages enrollment and command data within plain filesystem files and directories. It has zero dependencies and should run out of the box. The `-storage-dsn` flag specifies the filesystem directory for the database. The `file` backend has no storage options.

*Example:* `-storage file -storage-dsn /path/to/my/db`

#### mysql storage backend

* `-storage mysql`

Configures the MySQL storage backend. The `-storage-dsn` flag should be in the [format the SQL driver expects](https://github.com/go-sql-driver/mysql#dsn-data-source-name). Be sure to create your tables with the [schema.sql](../storage/mysql/schema.sql) file that corresponds to your NanoMDM version. Also make sure you apply any schema changes for each updated version (i.e. execute the numbered schema change files). MySQL 8.0.19 or later is required.

*Example:* `-storage mysql -storage-dsn nanomdm:nanomdm/mymdmdb`

Options are specified as a comma-separated list of "key=value" pairs. The mysql backend supports these options:

* `delete=1`, `delete=0`
  * This option turns on or off the command and response deleter. It is disabled by default. When enabled (with `delete=1`) command responses, queued commands, and commands themeselves will be deleted from the database after enrollments have responded to a command.

*Example:* `-storage mysql -storage-dsn nanomdm:nanomdm/mymdmdb -storage-options delete=1`

#### pgsql storage backend

* `-storage pgsql`

Configures the PostgreSQL storage backend. The `-storage-dsn` flag should be in the [format the SQL driver expects](https://pkg.go.dev/github.com/lib/pq#pkg-overview). Be sure to create your tables with the [schema.sql](../storage/pgsql/schema.sql) file that corresponds to your NanoMDM version. Also make sure you apply any schema changes for each updated version (i.e. execute the numbered schema change files). PostgreSQL 9.5 or later is required.

*Example:* `-storage pgsql -storage-dsn postgres://postgres:toor@localhost:5432/nanomdm`

Options are specified as a comma-separated list of "key=value" pairs. The pgsql backend supports these options:
* `delete=1`, `delete=0`
    * This option turns on or off the command and response deleter. It is disabled by default. When enabled (with `delete=1`) command responses, queued commands, and commands themselves will be deleted from the database after enrollments have responded to a command.

*Example:* `-storage pgsql -storage-dsn postgres://postgres:toor@localhost/nanomdm -storage-options delete=1`

#### multi-storage backend

You can configure multiple storage backends to be used simultaneously. Specifying multiple sets of `-storage`, `-storage-dsn`, & `-storage-options` flags will configure the "multi-storage" adapter. The flags must be specified in sets and are related to each other in the order they're specified: for example the first `-storage` flag corresponds to the first `-storage-dsn` flag and so forth.

Be aware that only the first storage backend will be "used" when interacting with the system, all other storage backends are called to, but any *results* are discarded. In other words consider them write-only. Also beware that you will have very bizaare results if you change to using multiple storage backends in the midst of existing enrollments. You will receive errors about missing database rows or data. A storage backend needs to be around when a device (or all devices) initially enroll(s). There is no "sync" or backfill system with multiple storage backends (see the migration ability if you need this).

The multi-storage backend is really only useful if you've always been using multiple storage backends or if you're doing some type of development or testing (perhaps creating a new storage backend).

For example to use both a `file` *and* `mysql` backend your command line might look like: `-storage file -storage-dsn db -storage mysql -storage-dsn nanomdm:nanomdm/mymdmdb`. You can also mix and match backends, or mutliple of the same backend. Behavior is undefined (and probably very bad) if you specify two backends of the same type with the same DSN.

### -dump

* dump MDM requests and responses to stdout

Dump MDM request bodies (i.e. complete Plist requests) to standard output for each request.

### -listen string

* HTTP listen address (default ":9000")

Specifies the listen address (interface & port number) for the server to listen on.

### -disable-mdm

* disable MDM HTTP endpoint

This switch disables MDM client capability. This effecitvely turns this running instance into "API-only" mode. It is not compatible with having an empty `-api` switch.

### -dm

* URL to send Declarative Management requests to

Specifies the "base" URL to send Declarative Management requests to. The full URL is constructed from this base URL appended with the type of Declarative Management ["Endpoint" request](https://developer.apple.com/documentation/devicemanagement/declarativemanagementrequest?language=objc) such as "status" or "declaration-items". Each HTTP request includes the NanoMDM enrollment ID as the HTTP header "X-Enrollment-ID". See [this blog post](https://micromdm.io/blog/wwdc21-declarative-management/) for more details.

Note that the URL should likely have a trailing slash. Otherwise path elements of the URL may to be cut off but by Golang's relative URL path resolver.

### -migration

* HTTP endpoint for enrollment migrations

NanoMDM supports a lossy form of MDM enrollment "migration." Essentially if a source MDM server can assemble enough of both Authenticate and TokenUpdate messages for an enrollment you can "migrate" enrollments by sending those Plist requests to the migration endpoint. Importantly this transfers the needed Push topic, token, and push magic to continue to send APNs push notifications to enrollments.

This switch turns on the migration endpoint.

### -retro

* Allow retroactive certificate-authorization association

By default NanoMDM disallows requests which did not have a certificate association setup in their Authenticate message. For new enrollments this is fine. However for enrollments that did not have a full Authenticate message (i.e. for enrollments that were migrated) they will lack such an association and be denied the ability to connect.

This switch turns on the ability for enrollments with no existing certificate association to create one, bypassing the authorization check. Note if an enrollment already has an association this will not overwrite it; only if no existing association exists.

### -version

* print version

Print version and exit.

### -webhook-url string

* URL to send requests to

NanoMDM supports a MicroMDM-compatible [webhook callback](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md) option. This switch turns on the webhook and specifies the URL.

### -auth-proxy-url string

* Reverse proxy URL target for MDM-authenticated HTTP requests

Enables the authentication proxy and reverse proxies HTTP requests from the server's `/authproxy/` endpoint to this URL if the client provides the device's enrollment authentication. See below for more information.

### -ua-zl-dc

* reply with zero-length DigestChallenge for UserAuthenticate

By default NanoMDM will respond to a `UserAuthenticate` message with an HTTP 410. This effectively declines management of that the user channel for that MDM user. Enabling this option turns on the "zero-length" Digest Challenge mode where NanoMDM replies with an empty Digest Challenge to enable management each time a client enrolls.

Note that the `UserAuthenticate` message is only for "directory" MDM users and not the "primary" MDM user enrollment. See also [Apple's discussion of UserAthenticate](https://developer.apple.com/documentation/devicemanagement/userauthenticate#discussion) for more information.

## HTTP endpoints & APIs

### MDM

* Endpoint: `/mdm`

The primary MDM endpoint is `/mdm` and needs to correspond to the `ServerURL` key in the enrollment profile. Both command & result handling as well as check-in handling happens on this endpoint by default. Note that if the `-checkin` switch is turned on then this endpoint will only handle command & result requests (having assumed that you updated your enrollment profile to include a separate `CheckInURL` key). Note the `-disable-mdm` switch will turn off this endpoint.

### MDM Check-in

* Endpoint: `/checkin`

The MDM check-in endpoint, if enabled, needs to correspond to the `CheckInURL` key in the enrollment profile. By default MDM check-ins are handled by the `/mdm` endpoint unless this switch is turned on in which case this endpoint handles them. This endpoint is disabled unless the `-checkin` switch is turned on. Note the `-disable-mdm` switch will turn off this endpoint.

### Push Cert

* Endpoint: `/v1/pushcert`

The push cert API endpoint allows for uploading an APNS push certificate. It takes a concatenated PEM-encoded APNs push certificate and private key as its HTTP body. Note the private key should not be encrypted. A quick way to utilize this endpoint is to use `curl`. For example:

```bash
$ cat /path/to/push.pem /path/to/push.key | curl -T - -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/pushcert'
{
	"topic": "com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9"
}
```

Here the `-T -` switch to `curl` tells it to take the standard-input and use it as the body for a PUT request to `/v1/pushcert`. We're also using `-u` to specify the API key (HTTP authentication). The server responded by telling us the topic that this Push certificate corresponds to.

### Push

* Endpoint: `/v1/push/`

The push API endpoint sends APNs push notifications to enrollments (which ask the MDM client to connect to the MDM server). This is a simple endpoint that takes enrollment IDs on the URL path:

```bash
$ curl -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8'
{
	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "8B16D295-AB2C-EAB9-90FF-8615C0DFBB08"
		}
	}
}
```

Here we successfully pushed to the client and received a push_result UUID from our push provider.

We can queue multiple pushes at the same time, too (note the separating comma in the URL):

```bash
$ curl -u nanomdm:nanomdm '[::1]:9000/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8,E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'
{
	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "5736F13F-E2A2-E8B9-E21C-3973BDAA4054"
		},
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "A70400AA-C5D8-DBA7-D66E-1296B36FA7F5"
		}
	}
}

```

### Enqueue

* Endpoint: `/v1/enqueue/`

The enqueue API endpoint allows sending of commands to enrollments. It takes a raw command Plist input as the HTTP body. The [`cmdr.py` script](../tools/cmdr.py) helps generate basic MDM commands. For example:

```bash
$ ./cmdr.py -r
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>ProfileList</string>
	</dict>
	<key>CommandUUID</key>
	<string>d7408b5d-f314-461f-bc5e-4ff107c03857</string>
</dict>
</plist>
```

(Note the `-r` switch here picks a random read-only MDM command)

Then, to submit a command to a NanoMDM enrollment:

```bash
$ ./cmdr.py -r | curl -T - -u nanomdm:nanomm 'http://127.0.0.1:9000/v1/enqueue/E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'
{
	"status": {
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "16C80450-B79F-E23B-F99B-0810179F244E"
		}
	},
	"command_uuid": "1ec2a267-1b32-4843-8ba0-2b06e80565c4",
	"request_type": "ProfileList"

```

Here we successfully queued a command to an enrollment ID (UDID) `E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8`  with command UUID `1ec2a267-1b32-4843-8ba0-2b06e80565c4` and we successfully sent a push request.

Note here, too, we can queue a command to multiple enrollments:

```bash
$ ./cmdr.py -r | curl -T - -u nanomdm:nanomm 'http://127.0.0.1:9000/v1/enqueue/99385AF6-44CB-5621-A678-A321F4D9A2C8,E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'

	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "4DE6E126-CC6C-37B2-7350-3AD1871C298F"
		},
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "7B9D73CD-186B-CCF4-D585-AEE9E8E4F0F3"
		}
	},
	"command_uuid": "9b7c63eb-14b4-4739-96b0-750a5c967371",
	"request_type": "ProvisioningProfileList"
}
```

Finally you can skip sending the push notification request by appending `?nopush=1` to the URI:

```bash
$ ./cmdr.py -r | curl -v -T - -u nanomdm:nanomdm '[::1]:9000/v1/enqueue/99385AF6-44CB-5621-A678-A321F4D9A2C8?nopush=1'
{
	"no_push": true,
	"command_uuid": "598544b5-b681-4ce2-8914-ba7f45ff5c02",
	"request_type": "CertificateList"
}
```

Of course the device won't check-in to retrieve this command, it will just sit in the queue until it is told to check-in using a push notification. This could be useful if you want to send a large number of commands and only want to push after the last command is sent.

### Migration

* Endpoint: `/migration`

The migration endpoint (as talked about above under the `-migration` switch) is an API endpoint that allows sending raw `TokenUpdate` and `Authenticate` messages to establish an enrollment — in particular the APNs push topic, token, and push magic. This endpoint bypasses certificate validation and certificate authentication (though still requires API HTTP authentication). In this way we enable a way to "migrate" MDM enrollments from another MDM. This is how the `llorne` tool of [the micro2nano project](https://github.com/micromdm/micro2nano) works, for example.

### Version

* Endpoint: `/version`

Returns a JSON response with the version of the running NanoMDM server.

### Authentication Proxy

* Endpoint: `/authproxy/`

If the `-auth-proxy-url` flag is provided then URLs that begin with `/authproxy/` will be reverse-proxied to the given target URL. Importantly this endpoint will authenticate the incoming request in the same way as other MDM endpoints (i.e. Check-In or Command Report and Response) — including whether we're using TLS client configuration or not (the `-cert-header` flag). Put together this allow you to have MDM-authenticated content retrieval.

This feature is ostensibly to support Declarative Device Management and in particular the ability for some "Asset" declarations to use "MDM" authentication for their content. For example the `com.apple.asset.data` declaration supports an [Authentication key](https://github.com/apple/device-management/blob/2bb1726786047949b5b1aa923be33b9ba0f83e37/declarative/declarations/assets/data.yaml#L40-L54) for configuring this ability.

As an example example if this feature is enabled and a request comes to the server as `/authproxy/foo/bar` and the `-auth-proxy-url` was set to, say, `http://[::1]:9008` then NanoMDM will reverse proxy this URL to `http://[::1]:9008/foo/bar`. An HTP 502 Bad Gateway response is sent back to the client for any issues proxying.

# Enrollment Migration (nano2nano)

The `nano2nano` tool extracts migration enrollment data from a given storage backend and sends it to a NanoMDM migration endpoint. In this way you can effectively migrate between database backends. For example if you started with a `file` backend you could migrate to a `mysql` backend and vice versa. Note that MDM servers must have *exactly* the same server URL for migrations to operate.

*Note:* Enrollment migration is **lossy**. It is not intended to bring over all data related to an enrollment — just the absolute bare minimum of data to support a migrated device being able to operate with MDM. For example previous commands & responses and even inventory data will be missing.

*Note:* There are some edge cases around enrollment migration. One such case is iOS unlock tokens. If the latest `TokenUpdate` did not contain the enroll-time unlock token for iOS then this information is probably lost in the migration. Again this feature is only meant to migrate the absolute minimum of information to allow for a device to be sent APNs push requests and have an operational command-queue.

## Switches

### -debug

* log debug messages

Enable additional debug logging.

### -storage, -storage-dsn, & -storage-options

See the "-storage, -storage-dsn, & -storage-options" section, above, for NanoMDM. The syntax and capabilities are the same.

### -key string

* NanoMDM API Key

The NanoMDM API key used to authenticate to the migration endpoint.

### -url string

* NanoMDM migration URL

The URL of the NanoMDM migration endpoint. For example "http://127.0.0.1:9000/migration".

### -version

* print version

Print version and exit.

## Example usage

```bash
$ ./nano2nano-darwin-amd64 -storage file -storage-dsn db -url 'http://127.0.0.1:9010/migration' -key nanomdm -debug
2021/06/04 14:29:54 level=info msg=storage setup storage=file
2021/06/04 14:29:54 level=info checkin=Authenticate device_id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
2021/06/04 14:29:54 level=info checkin=TokenUpdate device_id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
```
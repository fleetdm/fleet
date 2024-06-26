# NanoMDM Quickstart Guide

This quickstart guide is intended to quickly get a functioning NanoMDM instance up and running. This guide will use [ngrok](https://ngrok.com/) for easy setup and configuration of both public access & TLS. You are not required to use ngrok—this is merely a demonstration setup.

**Warning:** ngrok actively proxies live internet traffic to your computer. This means not only that it gets through firewalls and NATs but also that the ngrok service will have access to all proxied traffic it carries (i.e. anything sent to and from the MDM & SCEP servers). Check your security policy if this is allowable in your environment and beware.

This guide is intended to get NanoMDM *working* and *does not represent best practices* for running internet servers. Please don't run your production MDM service on ngrok! :)

## Requirements

- You'll need an MDM push certificate and private key. Documentation on how to attain one is out of scope for this document. You may refer to [MicroMDM's documentation](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/quickstart.md#configure-an-apns-certificate) for further information, but this guide assumes you have one at the ready. Note this is the _push_ certificate and _not_ the initial step of an _MDM CSR/vendor_ certificate.
- A posix-ish computer with a set of normal command-line tools available: `cat`, `curl`, Python 3, etc.
- Direct internet access (i.e. not proxied or outbound firewalled)

## SCEP server

If you don't already have a SCEP server we'll need to set one up. For this guide we'll setup a simple SCEP server using MicroMDM's SCEP server project. To get it visit the [GitHub releases page](https://github.com/micromdm/scep/releases) to fetch a pre-built binary.

*Note: If you already have a SCEP server, say, an existing MicroMDM instance, you can use that. Just be prepared to have the issuing CA certificate (and possibly intermediate certificates) on hand. NanoMDM will require it.*

### Download, extract and initialize SCEP server

```
$ mkdir scep && cd scep

$ curl -RLO https://github.com/micromdm/scep/releases/download/v2.1.0/scepserver-darwin-amd64-v2.1.0.zip
[snip]

$ unzip scepserver-darwin-amd64-v2.1.0.zip 
Archive:  scepserver-darwin-amd64-v2.1.0.zip
  inflating: scepserver-darwin-amd64  

$ ./scepserver-darwin-amd64 ca -init
Initializing new CA
```

By default the SCEP server will store CA & issuance data in a directory called `depot`.

### Run SCEP server

```
 ./scepserver-darwin-amd64 -allowrenew 0 -challenge nanomdm -debug
level=info ts=2021-05-29T21:19:40.902041Z caller=scepserver.go:163 transport=http address=:8080 msg=listening
```

This means the SCEP server is running. Make sure nothing else is listening on port 8080 before this. If there is you can change the port with the `-port` switch.

### Run ngrok for SCEP

Download `ngrok` from the [ngrok website](https://ngrok.com/) and run it to proxy the SCEP service:

```
$ ./ngrok http 8080
ngrok by @inconshreveable  (Ctrl+C to quit)

Session Status                online
Session Expires               1 hour, 59 minutes
Version                       2.3.40
Region                        United States (us)
Web Interface                 http://127.0.0.1:4040
Forwarding                    http://fd2a766cc645.ngrok.io -> http://localhost:8080
Forwarding                    https://fd2a766cc645.ngrok.io -> http://localhost:8080
[snip]
```

The "8080"  is the listen address (port) of the  running SCEP server, above. Note the URLs in the "Forwarding" section here (in our case here `https://fd2a766cc645.ngrok.io`). These are the public ngrok URLs that the SCEP service can be accessed using. You'll need this later.

*Note: the default (free) ngrok time limit is 2 hours. Your proxy connection (and URL) will end after that time. You may start another proxy/tunnel but note that your URLs will change each time.*

## NanoMDM

### Retrieve SCEP CA certificate(s)

Get a copy of your server's CA certificate. If you deployed the above SCEP server you can do this to save the CA:

```
$ curl 'https://fd2a766cc645.ngrok.io/scep?operation=GetCACert' | openssl x509 -inform DER > ca.pem 
```

This requests the CA certificate from the SCEP server, converts it into a PEM file and saves it to `ca.pem`.

### Download and extract NanoMDM

```
$ mkdir nanomdm && cd nanomdm

$ curl -RLO https://github.com/micromdm/nanomdm/releases/download/v0.2.0/nanomdm-darwin-amd64-v0.2.0.zip
[snip]

$ unzip nanomdm-darwin-amd64-v0.2.0.zip 
Archive:  nanomdm-darwin-amd64-v0.2.0.zip
  inflating: nanomdm-darwin-amd64  
```

### Run NanoMDM server

NanoMDM only requires the `-ca` switch to run which will authenticate the connecting MDM clients by validating their [device identity certificates](https://micromdm.io/blog/certificates/) against this CA certificate. This is the CA of the SCEP server that we saved, above.

We'll also supply the `-api` switch to set an API key and turn on API functionality. Here, we've set the API key to `nanomdm`.

```
./nanomdm-darwin-amd64 -ca ca.pem -api nanomdm -debug
2021/05/29 14:33:04 level=info msg=storage setup storage=file
2021/05/29 14:33:04 level=info msg=starting server listen=:9000
```

By default the file storage backend will write enrollment data into a directory called `db`.

*Note: API keys are simple HTTP Basic Authorization passwords with a username of "nanomdm". This means that any proxies, like ngrok, will have access to API authentication.* 

### Run (another) ngrok for NanoMDM

```
$ ./ngrok http 9000
ngrok by @inconshreveable  (Ctrl+C to quit)

Session Status                online
Session Expires               1 hour, 59 minutes
Version                       2.3.40
Region                        United States (us)
Web Interface                 http://127.0.0.1:4041
Forwarding                    http://625ae9460120.ngrok.io -> http://localhost:9000
Forwarding                    https://625ae9460120.ngrok.io -> http://localhost:9000
[snip]
```

The "9000" is the listen address (port) of the running NanoMDM server. Again, take note of the "forwarding" addresses.

If you get an error about being limited to 1 simultaneous ngrok agent, you can workaround this by launching both tunnels simultaneously via the `tunnels` stanza in the [ngrok agent config file](https://ngrok.com/docs/ngrok-agent/config#config-ngrok-tunnel-definitions). Add the following to your `ngrok.yml` config file, stop your ngrok agents, and then start them both with `ngrok start --all`.
```
tunnels:
  scep:
    proto: http
    addr: 8080
  nanomdm:
    proto: http
    addr: 9000
```

### Upload Push Certificate

To store the Push Certificate in NanoMDM we use the API:

```
$ cat /path/to/push.pem /path/to/push.key | curl -T - -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/pushcert'
{
	"topic": "com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9"
}
```

From the server logs you should also see something like:

```
2021/05/30 12:26:18 level=info handler=log addr=127.0.0.1 method=PUT path=/v1/pushcert agent=curl/7.54.0
2021/05/30 12:26:18 level=info handler=store-cert msg=stored push cert topic=com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9
```

This concatenates the certificate and private key PEM files with `cat` and then sends them to the "/v1/pushcert" endpoint using `curl`. Here we supplied the API key of "nanomdm" (and required username of nanomdm with the `-u` switch to `curl`). Note the push certificate private key needs to be unencrypted here. NanoMDM decodes the certificate and key, uploads them to storage, and returns the APNS "topic" that the push certificate contains. Keep note of this topic, you'll need it later.


## Configure enrollment profile

We'll need to author our enrollment profile for devices to know how to enroll in this MDM service. You can take a copy of the [example profile provided with NanoMDM](enroll.mobileconfig).

Make sure your enrollment profile contains the correct values for the SCEP payload URL as well as the MDM server URL. These will be from ngrok, above, If you followed this guide's instructions then those values would as follows. We also need to provide the SCEP challenge and MDM topic; also be from above. **Your values will be different, do not just copy/paste these values**:

* `URL` (in SCEP payload): `https://fd2a766cc645.ngrok.io/scep`
* `Challenge` (in SCEP payload): `nanomdm`
* `ServerURL` (in MDM payload): `https://625ae9460120.ngrok.io/mdm` (note the trailing `/mdm` here)
* `Topic`  (in MDM payload): `com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9`

## Enroll your machine!

WIth this modified enrollment profile you should now be able to enroll a device. Go ahead and do that—if its a Mac just double-click the (modified) `.mobileconfig` enrollment profile. If it worked you should see an `Authenticate` and `TokenUpdate` messages from NanoMDM:

```
2021/05/30 12:25:17 level=info handler=log addr=::1 method=PUT path=/mdm agent=MDM-OSX/1.0 mdmclient/1090 real_ip=1.2.3.4
2021/05/30 12:25:17 level=info service=certauth msg=cert associated enrollment=new id=99385AF6-44CB-5621-A678-A321F4D9A2C8 hash=87325e04e2c645032795ee0d74558ce52718bc472a14be4ea54cb92ba615ccc5
2021/05/30 12:25:17 level=info service=nanomdm msg=Authenticate id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device serial_number=C00DM2AKFD12
2021/05/30 12:25:18 level=info handler=log addr=::1 method=PUT path=/mdm agent=MDM-OSX/1.0 mdmclient/1090 real_ip=1.2.3.4
2021/05/30 12:25:18 level=info service=nanomdm msg=TokenUpdate id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
```

Here, `99385AF6-44CB-5621-A678-A321F4D9A2C8` is the enrollment ID which, for a device, is just the device's UDID. Note this, we'll want to use it, later.

## Send a Push Notification to the device

To send a push notification to the device asking it to check-in to our MDM service, we use the API:

```
$ curl -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8'
{
	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "8B16D295-AB2C-EAB9-90FF-8615C0DFBB08"
		}
	}
```

The `push_result` UUID (and absence of an error) is an indicator the push notification succeeded. Our push on the server should look like:

*Note: As an aside you can specify multiple enrollment IDs to send to by comma-separating them.*

```
2021/05/30 12:32:32 level=info handler=log addr=127.0.0.1 method=GET path=/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8 agent=curl/7.54.0
2021/05/30 12:32:32 level=info service=push msg=retrieved push cert topic=com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9
2021/05/30 12:32:32 level=debug handler=push msg=push count=1 errs=0
```

And, finally, if all worked well, the device should have checked into the server:

```
2021/05/30 12:32:33 level=info handler=log addr=::1 method=PUT path=/mdm agent=MDM-OSX/1.0 mdmclient/1090 real_ip=1.2.3.4
2021/05/30 12:32:33 level=info service=nanomdm status=Idle id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
2021/05/30 12:32:33 level=debug service=nanomdm msg=no command retrieved id=99385AF6-44CB-5621-A678-A321F4D9A2C8
```

*Note: the last debug line here is not a problem, it's just indicating that there was no queued command for this device to receive during its checkin.*

## Send a command

NanoMDM comes with a simple command generation tool, `cmdr.py` located here. You can download and execute it to get a feel for it's options: `./tools/cmdr.py`. Assuming you have it downloaded.

For example to generate a `SecurityInfo` just execute it with that command-line option:

```
$ ./tools/cmdr.py SecurityInfo
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>SecurityInfo</string>
	</dict>
	<key>CommandUUID</key>
	<string>d1b7fdda-52e1-45d1-80e6-8bf3b5d76f17</string>
</dict>
</plist>
```

It also has a `-r` mode to pick a random read-only command to generate. We'll use this to send a command to our MDM device!

```
$ ./tools/cmdr.py -r | curl -T - -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/enqueue/E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'
{
	"status": {
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "16C80450-B79F-E23B-F99B-0810179F244E"
		}
	},
	"command_uuid": "1ec2a267-1b32-4843-8ba0-2b06e80565c4",
	"request_type": "ProfileList"
```

On the server side we should see the command being queued and push notification sent.

*Note: we can specify a URL parameter of &no_push=1" to omit sending a push notification in the case of sending multiple commands.*

```
2021/05/30 12:46:39 level=info handler=log addr=127.0.0.1 method=PUT path=/v1/enqueue/99385AF6-44CB-5621-A678-A321F4D9A2C8 agent=curl/7.54.0
2021/05/30 12:46:39 level=debug handler=enqueue msg=enqueue command_uuid=1ec2a267-1b32-4843-8ba0-2b06e80565c4 request_type=ProfileList id_count=1 id_first=99385AF6-44CB-5621-A678-A321F4D9A2C8
2021/05/30 12:46:39 level=debug handler=enqueue msg=push count=1
```

And shortly thereafter we should see the client check-in, receive/de-queue/pop the command, and check-in a final time.

```
2021/05/30 12:46:40 level=info handler=log addr=::1 method=PUT path=/mdm agent=MDM-OSX/1.0 mdmclient/1090 real_ip=1.2.3.4
2021/05/30 12:46:40 level=info service=nanomdm status=Idle id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
2021/05/30 12:46:40 level=debug service=nanomdm msg=command retrieved id=99385AF6-44CB-5621-A678-A321F4D9A2C8 command_uuid=1ec2a267-1b32-4843-8ba0-2b06e80565c4
2021/05/30 12:46:40 level=info handler=log addr=::1 method=PUT path=/mdm agent=MDM-OSX/1.0 mdmclient/1090 real_ip=1.2.3.4
2021/05/30 12:46:40 level=info service=nanomdm status=Acknowledged id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device command_uuid=1ec2a267-1b32-4843-8ba0-2b06e80565c4
2021/05/30 12:46:40 level=debug service=nanomdm msg=no command retrieved id=E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8
```

This indiciates the full command->push->client check-in round trips all worked successfully.

## Setup a more production-ready service!

Great! You've verified the basics of getting NanoMDM going! Now you can move on to setting up a more production-ready service for actual real enrollments. This might include things like:

* A proper URL/domain name.
* A proper TLS certificate (possibly using Let's Encrypt, even).
* A proper reverse-proxy/load balancer (Nginx, HAProxy, Apache, etc.)
* A proper deployment, perhaps in a container environment like Docker, Kubernetes, etc. or even just running as a service with systemctl.
* More advanced configuration of NanoMDM including backend storage options, etc.

# scep

> The contents of this directory were copied (in February 2024) from https://github.com/fleetdm/scep (the `remove-path-setting-on-scep-handler` branch) which was forked from https://github.com/micromdm/scep.
> They were updated in September 2024 with changes up to github.com/micromdm/scep@781f8042a79cabcf61a5e6c01affdbadcb785932

[![CI](https://github.com/micromdm/scep/workflows/CI/badge.svg)](https://github.com/micromdm/scep/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/micromdm/scep/v2.svg)](https://pkg.go.dev/github.com/micromdm/scep/v2)

`scep` is a Simple Certificate Enrollment Protocol server and client

## Installation

Binary releases are available on the [releases page](https://github.com/micromdm/scep/releases).

### Compiling from source

To compile the SCEP client and server you will need [a Go compiler](https://golang.org/dl/) as well as standard tools like git, make, etc.

1. Clone the repository and get into the source directory: `git clone https://github.com/micromdm/scep.git && cd scep`
2. Compile the client and server binaries: `make` (for Windows: `make win`)

The binaries will be compiled in the current directory and named after the architecture. I.e. `scepclient-linux-amd64` and `scepserver-linux-amd64`.

### Docker

See Docker documentation below.

## Example setup

Minimal example for both server and client.

```
# SERVER:
# create a new CA
./scepserver-linux-amd64 ca -init
# start server
./scepserver-linux-amd64 -depot depot -port 2016 -challenge=secret

# SCEP request:
# in a separate terminal window, run a client
# note, if the client.key doesn't exist, the client will create a new rsa private key. Must be in PEM format.
./scepclient-linux-amd64 -private-key client.key -server-url=http://127.0.0.1:2016/scep -challenge=secret

# NDES request:
# note, this should point to an NDES server, scepserver does not provide NDES.
./scepclient-linux-amd64 -private-key client.key -server-url=https://scep.example.com:4321/certsrv/mscep/ -ca-fingerprint="e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
```

## Server Usage

The default flags configure and run the scep server.

`-depot` must be the path to a folder with `ca.pem` and `ca.key` files.  If you don't already have a CA to use, you can create one using the `ca` subcommand.

The scepserver provides one HTTP endpoint, `/scep`, that facilitates the normal PKIOperation/Message parameters.

Server usage:
```sh
$ ./scepserver-linux-amd64 -help
  -allowrenew string
    	do not allow renewal until n days before expiry, set to 0 to always allow (default "14")
  -capass string
    	passwd for the ca.key
  -challenge string
    	enforce a challenge password
  -crtvalid string
    	validity for new client certificates in days (default "365")
  -csrverifierexec string
    	will be passed the CSRs for verification
  -debug
    	enable debug logging
  -depot string
    	path to ca folder (default "depot")
  -log-json
    	output JSON logs
  -port string
    	port to listen on (default "8080")
  -version
    	prints version information
usage: scep [<command>] [<args>]
 ca <args> create/manage a CA
type <command> --help to see usage for each subcommand
```

Use the `ca -init` subcommand to create a new CA and private key.

CA sub-command usage:
```
$ ./scepserver-linux-amd64 ca -help
Usage of ca:
  -country string
    	country for CA cert (default "US")
  -depot string
    	path to ca folder (default "depot")
  -init
    	create a new CA
  -key-password string
    	password to store rsa key
  -keySize int
    	rsa key size (default 4096)
  -common_name string
        common name (CN) for CA cert (default "MICROMDM SCEP CA")
  -organization string
    	organization for CA cert (default "scep-ca")
  -organizational_unit string
    	organizational unit (OU) for CA cert (default "SCEP CA")
  -years int
    	default CA years (default 10)
```

### CSR verifier

The `-csrverifierexec` switch to the SCEP server allows for executing a command before a certificate is issued to verify the submitted CSR. Scripts exiting without errors (zero exit status) will proceed to certificate issuance, otherwise a SCEP error is generated to the client. For example if you wanted to just save the CSR this is a valid CSR verifier shell script:

```sh
#!/bin/sh

cat - > /tmp/scep.csr
```

## Client Usage

```sh
$ ./scepclient-linux-amd64 -help
Usage of ./scepclient-linux-amd64:
  -ca-fingerprint string
    	SHA-256 digest of CA certificate for NDES server. Note: Changed from MD5.
  -certificate string
    	certificate path, if there is no key, scepclient will create one
  -challenge string
    	enforce a challenge password
  -cn string
    	common name for certificate (default "scepclient")
  -country string
    	country code in certificate (default "US")
  -debug
    	enable debug logging
  -keySize int
    	rsa key size (default 2048)
  -locality string
    	locality for certificate
  -log-json
    	use JSON for log output
  -organization string
    	organization for cert (default "scep-client")
  -ou string
    	organizational unit for certificate (default "MDM")
  -private-key string
    	private key path, if there is no key, scepclient will create one
  -province string
    	province for certificate
  -server-url string
    	SCEP server url
  -version
    	prints version information
```

Note: Make sure to specify the desired endpoint in your `-server-url` value (e.g. `'http://scep.groob.io:2016/scep'`)

To obtain a certificate through Network Device Enrollment Service (NDES), set `-server-url` to a server that provides NDES.
This most likely uses the `/certsrv/mscep` path. You will need to add the `-ca-fingerprint` client argument during this request to specify which CA to use.

If you're not sure which SHA-256 hash (for a specific CA) to use, you can use the `-debug` flag to print them out for the CAs returned from the SCEP server.

## Docker

```sh
# first compile the Docker binaries
make docker

# build the image
docker build -t micromdm/scep:latest .

# create CA
docker run -it --rm -v /path/to/ca/folder:/depot micromdm/scep:latest ca -init

# run
docker run -it --rm -v /path/to/ca/folder:/depot -p 8080:8080 micromdm/scep:latest
```

## Server library

You can import the scep endpoint into another Go project. For an example take a look at [scepserver.go](cmd/scepserver/scepserver.go).

The SCEP server includes a built-in CA/certificate store. This is facilitated by the `Depot` and `CSRSigner` Go interfaces. This certificate storage to happen however you want. It also allows for swapping out the entire CA signer altogether or even using SCEP as a proxy for certificates.

#!/bin/bash

#
# This script generates test key material for mTLS in Orbit.
#
# This file has been committed to the repository to document
# how the key material was generated.
#
# Output files:
#  - client-ca.crt: This file is configured/used in the TLS server.
#  - client-ca.key: This file is not used in tests, but left in case
#    a new client certificate needs to be generated for testing.
#  - client.crt: This file is configured/used in a TLS client.
#  - client.key: This file is configured/used in a TLS client.

# Private key for the CA
openssl genrsa 2048 > client-ca.key

# Generate a CA certificate for the CA
openssl req -new -x509 -nodes -days 1000 -key client-ca.key > client-ca.crt

# Generate a client certificate signing request
openssl req -newkey rsa:2048 -days 398 -nodes -keyout client.key > client.req

# Have the CA sign the client certificate request and output the client certificate.
openssl x509 -req -in client.req -days 398 -CA client-ca.crt -CAkey client-ca.key -set_serial 01 > client.crt

rm client.req

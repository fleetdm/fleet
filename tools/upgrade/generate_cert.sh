#!/bin/bash

openssl req -x509 -newkey rsa:4096 -sha256 -days 3650 -nodes \
  -keyout fleet.key -out fleet.crt -extensions san -config \
  <(echo "[req]";
    echo distinguished_name=req;
    echo "[san]";
    echo subjectAltName=DNS:fleet-a,DNS:fleet-b
    ) \
  -subj "/CN=fleet"

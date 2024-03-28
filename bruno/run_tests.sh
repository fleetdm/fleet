#!/bin/bash

# write api token to disk
hurl hurl_test.hurl | jq '.token' | tr -d "\"" > apitoken

localtoken=$(cat apitoken)

hurl --variable=token=${localtoken} $1

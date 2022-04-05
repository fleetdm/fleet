#!/bin/bash

set -euo pipefail

# cd "$(dirname "${BASH_SOURCE[0]}")/.."
cd "$(dirname "${BASH_SOURCE[0]}")"

gen() {
    oapi-codegen -generate "types,client,chi-server,spec" openapi.bundled.yaml > openapi.gen.go
}

bundle() {
    openapi bundle openapi.yaml --output openapi.bundled.yaml
}

lint() {
    openapi lint openapi.yaml
}

serve() {
    openapi preview-docs -p 8081 openapi.yaml
}

"$@"

#!/bin/bash
#
# provision-tpm-network-certificate.sh
#
# PoC for https://github.com/fleetdm/fleet/issues/50541
#
# TPM-backed rewrite of the end-user network certificate script from
# https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#step-3-create-a-custom-script
#
# The original script generates an RSA key with `openssl genpkey` and leaves a
# (password-protected) private key on the filesystem. This version creates the
# private key INSIDE the TPM 2.0 chip instead, mirroring the key model Fleet
# already uses for host identity in ee/orbit/pkg/securehw:
#
#   - The key is created by the TPM itself (sensitiveDataOrigin), never by
#     software, so no code path ever handles the raw key material.
#   - fixedTPM and fixedParent are set, so the key can never be duplicated or
#     migrated to another TPM. Copying the on-disk artifacts to another host
#     yields a credential that cannot sign anything.
#   - The only thing persisted to disk is a TSS2 PEM keyfile: the public area
#     plus a private blob encrypted under the TPM's storage hierarchy. This is
#     the same format securehw writes via go-tpm-keyfiles.
#   - The CSR is signed via TPM2_Sign inside the chip (ECDSA), not by OpenSSL
#     in host memory.
#
# Instead of go-tpm, this script uses OpenSSL 3's tpm2 provider
# (https://github.com/tpm2-software/tpm2-openssl, Ubuntu package
# `tpm2-openssl`). Providers are the OpenSSL 3 replacement for the deprecated
# engine API, which matters because downstream consumers (wpa_supplicant,
# NetworkManager, VPN clients) will load the same provider to delegate their
# TLS client-auth signing to the TPM.
#
# There is deliberately NO software fallback: if the host has no usable TPM
# 2.0 device, the script fails with a clear message.
#
# Prerequisites on the host: openssl (>= 3.0), tpm2-openssl, curl, jq
#   apt-get install -y tpm2-openssl jq curl
#
# Fleet-side prerequisites (unchanged from the guide):
#   - API-only user with global maintainer role
#   - Fleet secret variable $FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN
#   - CA ID from GET /api/latest/fleet/certificate_authorities
#   - /opt/company/userinfo with USERNAME, TOKEN, CLIENT_ID (PASSWORD is no
#     longer needed -- the TPM enforces access, not a passphrase)

set -euo pipefail
umask 077

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

COMPANY_DIR="/opt/company"
USERINFO_FILE="${COMPANY_DIR}/userinfo"

# The TSS2 keyfile. Contains only the TPM-wrapped key blob -- useless without
# the TPM that created it. Consumers (wpa_supplicant / NetworkManager / VPN
# clients) reference this file through the tpm2 provider.
KEY_FILE="${COMPANY_DIR}/network-key.tss2.pem"
CERT_FILE="${COMPANY_DIR}/certificate.pem"

FLEET_URL="https://<Fleet-server-URL>"
CA_ID="<CA-ID>"
IDP_INTROSPECTION_URL="<IdP-introspection-URL>"

# securehw prefers P-384 and falls back to P-256 depending on TPM support;
# we mirror that below in generate_key(). RSA is intentionally not offered:
# the securehw plumbing this PoC follows is ECC-only, and ECDSA keeps the
# TPM signing operations fast during the EAP-TLS handshake.
CURVES=("P-384" "P-256")

log()  { echo "[tpm-network-cert] $*"; }
fail() { echo "[tpm-network-cert] ERROR: $*" >&2; exit 1; }

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# ---------------------------------------------------------------------------
# Preflight: require a TPM 2.0 and the OpenSSL tpm2 provider. No fallback.
# ---------------------------------------------------------------------------

preflight() {
    for tool in openssl curl jq; do
        command -v "${tool}" >/dev/null 2>&1 || fail "required tool '${tool}' is not installed"
    done

    # /dev/tpmrm0 is the kernel's TPM 2.0 resource manager device.
    if [[ ! -e /dev/tpmrm0 && ! -e /dev/tpm0 ]]; then
        fail "no TPM 2.0 device found (/dev/tpmrm0 or /dev/tpm0). This host cannot receive a hardware-backed network certificate, and software key storage is not permitted."
    fi

    case "$(openssl version)" in
        OpenSSL\ 3.*) ;;
        *) fail "OpenSSL 3.x is required for provider support; found: $(openssl version)" ;;
    esac

    # The tpm2 provider is what lets openssl (and later wpa_supplicant / VPN
    # clients) talk to the TPM. Ubuntu package: tpm2-openssl.
    if ! openssl list -providers -provider tpm2 -provider default >/dev/null 2>&1; then
        fail "OpenSSL tpm2 provider is not available. Install it with: apt-get install -y tpm2-openssl"
    fi

    [[ -f "${USERINFO_FILE}" ]] || fail "end user info file ${USERINFO_FILE} not found"
}

# ---------------------------------------------------------------------------
# Key generation inside the TPM
# ---------------------------------------------------------------------------

generate_key() {
    local curve tmp_key
    tmp_key="${WORK_DIR}/network-key.tss2.pem"

    for curve in "${CURVES[@]}"; do
        log "creating ECC ${curve} key inside the TPM"
        # The tpm2 provider's keygen template sets fixedTPM, fixedParent and
        # sensitiveDataOrigin -- the same attributes securehw sets in
        # CreateKey(). The private portion never leaves the chip; the output
        # file is a "TSS2 PRIVATE KEY" PEM holding the wrapped blob.
        if openssl genpkey -provider tpm2 -provider default \
            -algorithm EC -pkeyopt "group:${curve}" \
            -out "${tmp_key}" 2>"${WORK_DIR}/genpkey.err"; then
            log "TPM created a ${curve} key"
            mv "${tmp_key}" "${KEY_FILE}"
            chmod 600 "${KEY_FILE}"
            return 0
        fi
        log "TPM does not support ${curve} (or keygen failed), trying next curve"
    done

    cat "${WORK_DIR}/genpkey.err" >&2 || true
    fail "TPM key generation failed for all supported curves (${CURVES[*]})"
}

# Verify the hard requirement of the PoC: the key object must carry fixedTPM
# and fixedParent, and the on-disk artifact must be a wrapped TSS2 blob, not
# key material. Abort -- and remove the key -- if any of this doesn't hold.
verify_key() {
    local key_text

    grep -q "BEGIN TSS2 PRIVATE KEY" "${KEY_FILE}" \
        || fail "${KEY_FILE} is not a TSS2 keyfile; refusing to continue"

    # The tpm2 provider prints the object attributes of the underlying TPM
    # key when asked for a text dump.
    key_text="$(openssl pkey -provider tpm2 -provider default \
        -in "${KEY_FILE}" -noout -text 2>/dev/null)" \
        || fail "unable to load ${KEY_FILE} through the tpm2 provider (was it created on this TPM?)"

    local attr
    for attr in fixedTPM fixedParent sensitiveDataOrigin; do
        if ! grep -qi "${attr}" <<<"${key_text}"; then
            rm -f "${KEY_FILE}"
            fail "TPM key is missing required attribute '${attr}'; key deleted, aborting"
        fi
    done

    log "verified key attributes: fixedTPM, fixedParent, sensitiveDataOrigin"
}

# Reuse an existing TPM key on renewal so the host keeps the same identity
# across certificate rotations and reboots; only mint a new one if the file
# is missing or no longer loads on this TPM.
ensure_key() {
    if [[ -f "${KEY_FILE}" ]] \
        && openssl pkey -provider tpm2 -provider default -in "${KEY_FILE}" -noout >/dev/null 2>&1; then
        log "reusing existing TPM key at ${KEY_FILE}"
    else
        [[ -f "${KEY_FILE}" ]] && log "existing keyfile no longer loads on this TPM; recreating"
        generate_key
    fi
    verify_key
}

# ---------------------------------------------------------------------------
# CSR (signed inside the TPM) and certificate request to Fleet
# ---------------------------------------------------------------------------

request_certificate() {
    local csr_file="${WORK_DIR}/network.csr"
    local response="${WORK_DIR}/response.json"

    # shellcheck disable=SC1090
    . "${USERINFO_FILE}"

    log "generating CSR for ${USERNAME} (ECDSA signature performed inside the TPM)"
    openssl req -new -sha256 \
        -provider tpm2 -provider default \
        -key "${KEY_FILE}" \
        -subj "/CN=CustomerUserNetworkAccess:${USERNAME}" \
        -addext "subjectAltName=DNS:example.com, email:${USERNAME}, otherName:msUPN;UTF8:${USERNAME}" \
        -out "${csr_file}"

    log "requesting certificate from Fleet"
    jq -n \
        --rawfile csr "${csr_file}" \
        --arg url "${IDP_INTROSPECTION_URL}" \
        --arg token "${TOKEN}" \
        --arg client_id "${CLIENT_ID}" \
        '{csr: $csr, idp_oauth_url: $url, idp_token: $token, idp_client_id: $client_id, return_pem_certificate: true}' \
        >"${WORK_DIR}/request.json"

    local http_code
    http_code="$(curl -sS -o "${response}" -w '%{http_code}' \
        "${FLEET_URL}/api/latest/fleet/certificate_authorities/${CA_ID}/request_certificate" \
        -X POST \
        -H 'accept: application/json, text/plain, */*' \
        -H "authorization: Bearer ${FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN}" \
        -H 'content-type: application/json' \
        --data-binary "@${WORK_DIR}/request.json")"

    [[ "${http_code}" == "200" ]] \
        || fail "certificate request failed (HTTP ${http_code}): $(cat "${response}")"

    jq -re .certificate "${response}" >"${CERT_FILE}.tmp" \
        || fail "no certificate in Fleet response: $(cat "${response}")"
    mv "${CERT_FILE}.tmp" "${CERT_FILE}"
    chmod 644 "${CERT_FILE}"
    log "certificate written to ${CERT_FILE}"
}

# Confirm the issued certificate actually belongs to the TPM-resident key.
verify_certificate() {
    local cert_pub key_pub
    cert_pub="$(openssl x509 -in "${CERT_FILE}" -pubkey -noout)"
    key_pub="$(openssl pkey -provider tpm2 -provider default -in "${KEY_FILE}" -pubout 2>/dev/null)"
    [[ "${cert_pub}" == "${key_pub}" ]] \
        || fail "issued certificate public key does not match the TPM key"
    log "certificate public key matches the TPM-resident private key"
}

preflight
mkdir -p "${COMPANY_DIR}"
ensure_key
request_certificate
verify_certificate

log "done. Private key never existed outside the TPM; only the wrapped TSS2 blob is on disk."

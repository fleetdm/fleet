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
# Usage:
#   provision-tpm-network-certificate.sh            renew (default)
#   provision-tpm-network-certificate.sh --rotate   explicitly rotate the key
#   ROTATE_KEY=1 provision-tpm-network-certificate.sh   (same as --rotate)
#
# Key lifecycle safety model:
#
#   Renewal (default, no flag):
#     - An existing, loadable TPM key is REUSED so the host keeps the same
#       identity across certificate rotations and reboots.
#     - If the existing key CANNOT be loaded, the script FAILS and leaves the
#       keyfile and the installed certificate exactly as they were. A load
#       failure may be transient (TPM busy, provider/driver error), and the
#       installed certificate was issued for THAT key: replacing the key
#       first and then failing later (CSR generation, enrollment) would
#       orphan the installed certificate -- its private key would simply be
#       gone.
#     - A MISSING keyfile is unambiguous (nothing can sign anymore): a new
#       key is generated and verified at a temporary path, and only then
#       installed.
#     - A freshly issued certificate is likewise staged at a temporary path
#       and validated against the key before it replaces the installed
#       certificate; the previous certificate is kept as certificate.pem.prev.
#
#   Rotation (--rotate / ROTATE_KEY=1): the ONLY path that replaces an
#   installed key. It generates and verifies a candidate key at a separate
#   path, requests and validates a certificate for the candidate, and only
#   then activates the new key+certificate together, retaining
#   network-key.tss2.pem.prev / certificate.pem.prev rollback copies.
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
# Key lifecycle: generation, verification, safe renewal, explicit rotation
# ---------------------------------------------------------------------------

# Create an ECC key inside the TPM and move the resulting TSS2 keyfile to
# $1. The keygen output always lands in WORK_DIR first; the move is the only
# step that exposes it, and callers verify the candidate before it is ever
# used to sign or request anything.
generate_key() {
    local out_file="${1:?usage: generate_key <out_file>}"
    local curve tmp_key="${WORK_DIR}/.keygen.tss2.pem"

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
            mv "${tmp_key}" "${out_file}"
            chmod 600 "${out_file}"
            return 0
        fi
        log "TPM does not support ${curve} (or keygen failed), trying next curve"
    done

    cat "${WORK_DIR}/genpkey.err" >&2 || true
    fail "TPM key generation failed for all supported curves (${CURVES[*]})"
}

# Reject a key that failed verification. Deletes the file only when
# cleanup=1 -- i.e. only for a key this script itself just generated, which
# nothing else can reference. An installed key is never deleted by this
# script: the installed certificate was issued for it, and deleting the key
# would orphan that certificate.
reject_key() {
    local key_file="$1"
    local cleanup="$2"
    local reason="$3"
    if [[ "${cleanup}" == "1" ]]; then
        rm -f "${key_file}"
    fi
    fail "${reason}"
}

# Verify the hard requirement of the PoC: the key object must carry fixedTPM
# and fixedParent, and the on-disk artifact must be a wrapped TSS2 blob, not
# key material. $2 (cleanup) controls whether a failing key is removed; see
# reject_key() for why an installed key is never deleted.
verify_key() {
    local key_file="$1"
    local cleanup="${2:-0}"
    local key_text attr

    grep -q "BEGIN TSS2 PRIVATE KEY" "${key_file}" \
        || reject_key "${key_file}" "${cleanup}" \
            "${key_file} is not a TSS2 keyfile; refusing to continue"

    # The tpm2 provider prints the object attributes of the underlying TPM
    # key when asked for a text dump.
    if ! key_text="$(openssl pkey -provider tpm2 -provider default \
            -in "${key_file}" -noout -text 2>/dev/null)"; then
        reject_key "${key_file}" "${cleanup}" \
            "unable to load ${key_file} through the tpm2 provider (was it created on this TPM?)"
    fi

    for attr in fixedTPM fixedParent sensitiveDataOrigin; do
        if ! grep -qi "${attr}" <<<"${key_text}"; then
            reject_key "${key_file}" "${cleanup}" \
                "TPM key ${key_file} is missing required attribute '${attr}'"
        fi
    done

    log "verified key attributes of ${key_file}: fixedTPM, fixedParent, sensitiveDataOrigin"
}

# Decide which key to use for this run.
#
#   - Existing key that LOADS: reuse it (stable identity across renewals).
#   - Existing key that DOES NOT load: fail, and touch nothing. The failure
#     may be transient, and the installed certificate at ${CERT_FILE} was
#     issued for exactly this key -- replacing the key now and then failing
#     later (CSR generation, enrollment) would leave the host with a
#     certificate whose private key no longer exists.
#   - Missing key: generate a candidate, verify it, then install it.
ensure_key() {
    local candidate="${WORK_DIR}/candidate-key.tss2.pem"

    if [[ -f "${KEY_FILE}" ]]; then
        local load_err
        if load_err="$(openssl pkey -provider tpm2 -provider default \
                -in "${KEY_FILE}" -noout 2>&1)"; then
            log "reusing existing TPM key at ${KEY_FILE}"
        else
            if [[ -n "${load_err}" ]]; then
                log "key load error: ${load_err}"
            fi
            log "refusing to replace the installed key: the certificate at ${CERT_FILE} was issued for it, and replacing the key before a later failure would orphan that certificate"
            fail "cannot load the existing TPM key at ${KEY_FILE}; the keyfile and the installed certificate were left untouched. If the failure is transient (TPM busy, provider/driver error, TPM state after a reboot), resolve it and re-run this script. If you deliberately need a NEW key+certificate identity (e.g. the TPM was replaced), re-run with --rotate (or ROTATE_KEY=1): it generates and verifies a candidate key at a separate path, obtains and validates its certificate, and only then activates the pair, retaining .prev rollback copies."
        fi
        verify_key "${KEY_FILE}" 0
    else
        if [[ -f "${CERT_FILE}" ]]; then
            log "warning: certificate at ${CERT_FILE} exists but no key does; a new key means a new identity will replace it once enrollment succeeds"
        fi
        log "no keyfile at ${KEY_FILE}; generating a new TPM key"
        generate_key "${candidate}"
        verify_key "${candidate}" 1
        mv "${candidate}" "${KEY_FILE}"
        log "installed new TPM key at ${KEY_FILE}"
    fi
}

# Install a fully verified candidate certificate over the live one,
# retaining the previous certificate as ${CERT_FILE}.prev (nothing is ever
# deleted). The move is a rename/copy of a complete file, so the live path
# is never observed in a half-written state.
install_certificate() {
    local candidate_cert="$1"

    if [[ -f "${CERT_FILE}" ]]; then
        cp -p "${CERT_FILE}" "${CERT_FILE}.prev"
        chmod 644 "${CERT_FILE}.prev"
        log "previous certificate retained at ${CERT_FILE}.prev"
    fi

    mv "${candidate_cert}" "${CERT_FILE}"
    chmod 644 "${CERT_FILE}"
    log "installed certificate at ${CERT_FILE}"
}

# Explicit rotation (--rotate): the only path that replaces an installed
# key. The live key and certificate files are untouched until the
# replacement pair has been fully verified, after which it is activated as a
# unit with rollback copies retained.
rotate_key_and_certificate() {
    local candidate_key="${WORK_DIR}/candidate-key.tss2.pem"
    local candidate_cert="${WORK_DIR}/candidate-certificate.pem"

    log "explicit rotation: generating a candidate key at a separate path (installed key stays in place until the new pair is verified)"
    generate_key "${candidate_key}"
    verify_key "${candidate_key}" 1

    log "explicit rotation: requesting a certificate for the candidate key (installed certificate stays in place)"
    request_certificate "${candidate_key}" "${candidate_cert}"
    verify_certificate "${candidate_cert}" "${candidate_key}"

    activate_key_and_certificate "${candidate_key}" "${candidate_cert}"
}

# Promote a fully verified candidate key+certificate to their installed
# locations. The previous installed key and certificate are first retained as
# .prev rollback copies -- nothing is ever deleted -- and the verified pair is
# moved into place. The two moves run back-to-back; the only transient
# mismatch window is the instant between them, and an interrupted run can be
# rolled back from the .prev copies.
activate_key_and_certificate() {
    local candidate_key="$1"
    local candidate_cert="$2"
    
    # Step 1: Prepare rollback copies for both key and cert
    if [[ -f "${KEY_FILE}" ]]; then
        cp -p "${KEY_FILE}" "${KEY_FILE}.prev"
        chmod 600 "${KEY_FILE}.prev"
        log "previous key retained at ${KEY_FILE}.prev"
    fi
    
    if [[ -f "${CERT_FILE}" ]]; then
        cp -p "${CERT_FILE}" "${CERT_FILE}.prev"
        chmod 644 "${CERT_FILE}.prev"
        log "previous certificate retained at ${CERT_FILE}.prev"
    fi

    # Step 2: Atomic moves
    mv "${candidate_key}" "${KEY_FILE}"
    chmod 600 "${KEY_FILE}"
    
    mv "${candidate_cert}" "${CERT_FILE}"
    chmod 644 "${CERT_FILE}"
    log "activated new key+certificate pair"
}

# ---------------------------------------------------------------------------
# CSR (signed inside the TPM) and certificate request to Fleet
# ---------------------------------------------------------------------------

# Sign a CSR with the given key (the ECDSA signature is performed inside the
# TPM), request a certificate from Fleet, and write it to the given path.
# Callers pass candidate paths inside WORK_DIR and install the result only
# after validation (see install_certificate).
request_certificate() {
    local key_file="$1"
    local cert_file="$2"
    local csr_file="${WORK_DIR}/network.csr"
    local response="${WORK_DIR}/response.json"

    # shellcheck disable=SC1090
    . "${USERINFO_FILE}"

    log "generating CSR for ${USERNAME} (ECDSA signature performed inside the TPM)"
    openssl req -new -sha256 \
        -provider tpm2 -provider default \
        -key "${key_file}" \
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

    jq -re .certificate "${response}" >"${cert_file}.tmp" \
        || fail "no certificate in Fleet response: $(cat "${response}")"
    mv "${cert_file}.tmp" "${cert_file}"
    chmod 644 "${cert_file}"
    log "certificate written to ${cert_file}"
}

# Confirm the issued certificate actually belongs to the given key.
verify_certificate() {
    local cert_file="$1"
    local key_file="$2"
    local cert_pub key_pub
    cert_pub="$(openssl x509 -in "${cert_file}" -pubkey -noout)"
    key_pub="$(openssl pkey -provider tpm2 -provider default -in "${key_file}" -pubout 2>/dev/null)"
    [[ "${cert_pub}" == "${key_pub}" ]] \
        || fail "certificate ${cert_file} public key does not match key ${key_file}"
    log "certificate public key matches the TPM-resident private key"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

usage() {
    cat <<EOF
Usage: $(basename "$0") [--rotate]

  (no flag)  Renew: reuse the installed TPM key and install a new
             certificate for it. Never replaces the installed key; if the
             installed key cannot be loaded, fails and leaves everything
             in place.

  --rotate   Explicitly replace the installed key+certificate pair:
             generate and verify a candidate key, obtain and validate a
             certificate for it, then activate the pair, retaining .prev
             rollback copies of the previous key and certificate.

The ROTATE_KEY=1 environment variable is equivalent to --rotate.
EOF
}

# ROTATE_KEY defaults to 0; the environment can set it so the same script
# file can be driven from a Fleet script policy.
ROTATE_KEY="${ROTATE_KEY:-0}"
for arg in "$@"; do
    case "${arg}" in
        --rotate)
            ROTATE_KEY="1"
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            fail "unknown argument '${arg}' (supported: --rotate)"
            ;;
    esac
done

preflight
mkdir -p "${COMPANY_DIR}"

if [[ "${ROTATE_KEY}" == "1" ]]; then
    if [[ -f "${KEY_FILE}" ]]; then
        log "rotating installed key+certificate pair"
    else
        log "no installed key found; nothing to rotate -- provisioning a fresh key+certificate pair"
    fi
    rotate_key_and_certificate
else
    ensure_key
    # Stage the freshly issued certificate at a temporary path and validate
    # it against the key before it replaces the installed certificate.
    candidate_cert="${WORK_DIR}/candidate-certificate.pem"
    request_certificate "${KEY_FILE}" "${candidate_cert}"
    verify_certificate "${candidate_cert}" "${KEY_FILE}"
    install_certificate "${candidate_cert}"
fi

log "done. Private key never existed outside the TPM; only the wrapped TSS2 blob is on disk."

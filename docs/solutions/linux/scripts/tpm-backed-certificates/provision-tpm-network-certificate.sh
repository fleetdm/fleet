#!/bin/bash
#
# provision-tpm-network-certificate.sh
#
# Requests an end-user network certificate from Fleet for a private key that
# is created inside the host's TPM 2.0 and cannot be exported.
#
#   - The TPM creates the key with fixedTPM, fixedParent and
#     sensitiveDataOrigin set. Only the TPM-wrapped TSS2 keyfile is written
#     to disk, and it only loads on the TPM that created it.
#   - The CSR is signed inside the TPM and sent to Fleet's
#     request_certificate endpoint. The issued certificate is checked
#     against the TPM key before it is installed.
#   - There is no software fallback. Without a usable TPM, the script fails.
#
# Modes:
#   (default)  Renew. Reuse the installed key (or create one if none
#              exists) and install a new certificate for it.
#   --rotate   Replace the installed key and certificate with a new pair.
#
# Requires: openssl 3, tpm2-openssl, curl, jq, flock.
# Reads USERNAME, TOKEN and CLIENT_ID from /opt/company/userinfo.

set -euo pipefail
umask 077

COMPANY_DIR="/opt/company"
USERINFO_FILE="${COMPANY_DIR}/userinfo"
LOCK_FILE="${COMPANY_DIR}/provision.lock"
mkdir -p "${COMPANY_DIR}"

KEY_FILE="${COMPANY_DIR}/network-key.tss2.pem"
CERT_FILE="${COMPANY_DIR}/certificate.pem"

FLEET_URL="https://<Fleet-server-URL>"
# ID of the CA from GET /api/latest/fleet/certificate_authorities.
CA_ID="<CA-ID>"
IDP_INTROSPECTION_URL="<IdP-introspection-URL>"

# Tried in order. Not every TPM supports P-384.
CURVES=("P-384" "P-256")

log()  { echo "[tpm-network-cert] $*"; }
fail() { echo "[tpm-network-cert] ERROR: $*" >&2; exit 1; }

# Scratch space on the same filesystem as COMPANY_DIR, so moving files into
# place is an atomic rename.
WORK_DIR="$(mktemp -d -p "${COMPANY_DIR}")"
trap 'rm -rf "${WORK_DIR}"' EXIT

# Fail unless the host has the tools, a TPM 2.0, a working tpm2 provider and
# the userinfo file.
preflight() {
    for tool in openssl curl jq flock; do
        command -v "${tool}" >/dev/null 2>&1 || fail "required tool '${tool}' is not installed"
    done

    if [[ ! -e /dev/tpmrm0 && ! -e /dev/tpm0 ]]; then
        fail "no TPM 2.0 device found (/dev/tpmrm0 or /dev/tpm0). This host cannot receive a hardware-backed network certificate, and software key storage is not permitted."
    fi

    case "$(openssl version)" in
        OpenSSL\ 3.*) ;;
        *) fail "OpenSSL 3.x is required for provider support; found: $(openssl version)" ;;
    esac

    # Loading the provider also opens a TPM connection. TPM2OPENSSL_TCTI
    # overrides which TPM interface it uses.
    if ! openssl list -providers -provider tpm2 -provider default >/dev/null 2>&1; then
        fail "OpenSSL tpm2 provider could not be loaded or could not reach the TPM. Install the provider with: apt-get install -y tpm2-openssl"
    fi

    [[ -f "${USERINFO_FILE}" ]] || fail "end user info file ${USERINFO_FILE} not found"
}

# Create an ECC key inside the TPM and write its TSS2 keyfile to $1.
generate_key() {
    local out_file="${1:?usage: generate_key <out_file>}"
    local curve tmp_key="${WORK_DIR}/.keygen.tss2.pem"

    for curve in "${CURVES[@]}"; do
        log "creating ECC ${curve} key inside the TPM"
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

# Fail on a key that did not verify. Delete it only when cleanup=1, i.e. a
# key this run just created. An installed key is never deleted.
reject_key() {
    local key_file="$1"
    local cleanup="$2"
    local reason="$3"
    if [[ "${cleanup}" == "1" ]]; then
        rm -f "${key_file}"
    fi
    fail "${reason}"
}

# Fail unless $1 is a TSS2 keyfile for a TPM key with fixedTPM, fixedParent
# and sensitiveDataOrigin set.
verify_key() {
    local key_file="$1"
    local cleanup="${2:-0}"
    local key_text attr

    grep -q "BEGIN TSS2 PRIVATE KEY" "${key_file}" \
        || reject_key "${key_file}" "${cleanup}" \
            "${key_file} is not a TSS2 keyfile; refusing to continue"

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

# Renew: reuse the installed key if it loads. If it exists but does not
# load, fail without changing anything, since the installed certificate
# belongs to it. If no key exists, create and verify one.
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

# Install a verified certificate, keeping the previous one as .prev.
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

# Rotate: create and verify a new key, request and verify its certificate,
# then activate both. The installed pair is untouched until then.
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

# Move a verified key and certificate into place, keeping the previous pair
# as .prev. If the certificate move fails, roll the key back so the live
# key and certificate still match.
activate_key_and_certificate() {
    local candidate_key="$1"
    local candidate_cert="$2"
    local had_key=0
    local restore_key="${WORK_DIR}/restore-key.tss2.pem"

    if [[ -f "${KEY_FILE}" ]]; then
        cp -p "${KEY_FILE}" "${KEY_FILE}.prev"
        chmod 600 "${KEY_FILE}.prev"
        had_key=1
        log "previous key retained at ${KEY_FILE}.prev"
    fi

    if [[ -f "${CERT_FILE}" ]]; then
        cp -p "${CERT_FILE}" "${CERT_FILE}.prev"
        chmod 644 "${CERT_FILE}.prev"
        log "previous certificate retained at ${CERT_FILE}.prev"
    fi

    mv "${candidate_key}" "${KEY_FILE}" \
        || fail "key activation failed; the installed key and certificate were not changed"

    if ! mv "${candidate_cert}" "${CERT_FILE}"; then
        if [[ "${had_key}" == "1" ]]; then
            if ! { cp -p "${KEY_FILE}.prev" "${restore_key}" && mv "${restore_key}" "${KEY_FILE}"; }; then
                fail "certificate activation failed AND restoring the previous key failed; copy ${KEY_FILE}.prev back to ${KEY_FILE} by hand"
            fi
        else
            rm -f "${KEY_FILE}" \
                || fail "certificate activation failed AND removing the new key failed; delete ${KEY_FILE} by hand"
        fi
        fail "certificate activation failed; the key was rolled back and the installed certificate was not changed"
    fi

    log "activated new key+certificate pair"
}

# Create a CSR signed by the TPM key in $1, request a certificate from
# Fleet, and write it to $2.
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

    # The token goes in a config file so it does not appear in the process
    # list. Fleet fills in the FLEET_SECRET_ variable before the script runs.
    local http_code
    local curl_config="${WORK_DIR}/curl.cfg"
    printf 'header = "authorization: Bearer %s"\n' "${FLEET_SECRET_REQUEST_CERTIFICATE_API_TOKEN}" > "${curl_config}"
    chmod 600 "${curl_config}"

    http_code="$(curl --connect-timeout 15 --max-time 60 -sS -o "${response}" -w '%{http_code}' --config "${curl_config}" \
        "${FLEET_URL}/api/latest/fleet/certificate_authorities/${CA_ID}/request_certificate" \
        -X POST \
        -H 'accept: application/json, text/plain, */*' \
        -H 'content-type: application/json' \
        --data-binary "@${WORK_DIR}/request.json")"

    if [[ "${http_code}" == "200" ]]; then
        :
    elif [[ "${http_code}" == "0" ]]; then
        fail "certificate request failed: network error or timeout (HTTP 0)"
    else
        fail "certificate request failed (HTTP ${http_code})"
    fi

    jq -re .certificate "${response}" >"${cert_file}.tmp" \
        || fail "certificate request response does not contain a certificate"
    mv "${cert_file}.tmp" "${cert_file}"
    chmod 644 "${cert_file}"
    log "certificate written to ${cert_file}"
}

# Fail unless the certificate in $1 was issued for the TPM key in $2.
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

# 1 = rotate. Fleet runs scripts without arguments or environment
# variables, so rotating from Fleet needs a copy with this set to 1.
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
(
    # One run at a time, so overlapping runs cannot mismatch key and certificate.
    flock -w 120 -x 200 || fail "another run still holds ${LOCK_FILE} after 120s"
    if [[ "${ROTATE_KEY}" == "1" ]]; then
        if [[ -f "${KEY_FILE}" ]]; then
            log "rotating installed key+certificate pair"
        else
            log "no installed key found; nothing to rotate -- provisioning a fresh key+certificate pair"
        fi
        rotate_key_and_certificate
    else
        ensure_key
        candidate_cert="${WORK_DIR}/candidate-certificate.pem"
        request_certificate "${KEY_FILE}" "${candidate_cert}"
        verify_certificate "${candidate_cert}" "${KEY_FILE}"
        install_certificate "${candidate_cert}"
    fi
) 200>"${LOCK_FILE}"

log "done. Private key never existed outside the TPM; only the wrapped TSS2 blob is on disk."

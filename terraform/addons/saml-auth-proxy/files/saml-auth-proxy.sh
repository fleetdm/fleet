mkdir -p $(dirname ${SAML_PROXY_SP_CERT_PATH:?})
mkdir -p $(dirname ${SAML_PROXY_SP_KEY_PATH:?})
echo "${SAML_PROXY_SP_CERT_BYTES:?}" > "${SAML_PROXY_SP_CERT_PATH:?}"
echo "${SAML_PROXY_SP_KEY_BYTES:?}" > "${SAML_PROXY_SP_KEY_PATH:?}"
/usr/bin/saml-auth-proxy

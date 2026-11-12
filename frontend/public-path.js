import URL_PREFIX from "router/url_prefix";
import getCSPNonce from "utilities/nonce";

// Sets the path used to load assets
__webpack_public_path__ = `${URL_PREFIX}/assets/`; // eslint-disable-line camelcase, no-undef

// Stamp the CSP nonce on the <script> and <link> tags webpack injects to load
// lazy chunks. Must be set before any chunk loads, so it lives here in the
// entry's first import. No-op when the page is served without a CSP.
__webpack_nonce__ = getCSPNonce(); // eslint-disable-line camelcase, no-undef

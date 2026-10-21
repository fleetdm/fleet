const { OAuth2Client } = require("google-auth-library");

const CHAT_ISSUER = "chat@system.gserviceaccount.com";
const CHAT_CERTS_URL = `https://www.googleapis.com/service_accounts/v1/metadata/x509/${CHAT_ISSUER}`;
const CERTS_TTL_MS = 60 * 60 * 1000;
const CERTS_FETCH_TIMEOUT_MS = 10 * 1000;

// google-auth-library error messages can embed the token after a colon.
const reasonOf = (err) => String((err && err.message) || err).split(":")[0];

/**
 * Build a function that checks the bearer token Google Chat sends with each
 * event. Two token types are accepted, and both are bound to our Google Cloud
 * project:
 *
 * - Add-on Chat apps (the console default): a Google ID token for this
 *   project's add-on service agent, with aud = the endpoint URL.
 * - Classic Chat apps with the "Project number" authentication audience: a JWT
 *   from chat@system.gserviceaccount.com with aud = the project number.
 *
 * The classic "HTTP endpoint URL" audience is deliberately not accepted: those
 * tokens identify only the shared Chat service account and the URL, so a Chat
 * app in any other project pointed at our URL would pass.
 *
 * Resolves to { ok, reason }. Throws only if Chat's signing certs can't be
 * fetched, so outages surface as 5xx (which Chat retries) rather than 401.
 */
function createRequestVerifier({ projectNumber, endpointUrl }, { client = new OAuth2Client(), fetchCerts = fetchChatCerts } = {}) {
  // google-auth-library skips the audience check when it's undefined.
  if (!projectNumber || !endpointUrl) {
    throw new Error("Google Chat request verification needs both a project number and an endpoint URL");
  }
  const addOnServiceAgent = `service-${projectNumber}@gcp-sa-gsuiteaddons.iam.gserviceaccount.com`;
  let certs = null;
  let certsFetchedAt = 0;

  const chatCerts = async () => {
    if (!certs || Date.now() - certsFetchedAt > CERTS_TTL_MS) {
      certs = await fetchCerts();
      certsFetchedAt = Date.now();
    }
    return certs;
  };

  return async function verify(authorizationHeader) {
    const match = /^Bearer (\S+)$/.exec(authorizationHeader || "");
    if (!match) return { ok: false, reason: "missing bearer token" };
    const token = match[1];

    let addOnReason;
    try {
      const ticket = await client.verifyIdToken({ idToken: token, audience: endpointUrl });
      const payload = ticket.getPayload();
      if (payload.email_verified && payload.email === addOnServiceAgent) return { ok: true };
      addOnReason = `token is for ${payload.email}, expected ${addOnServiceAgent}`;
    } catch (err) {
      // Not an add-on token for this endpoint; try the classic format.
      addOnReason = reasonOf(err);
    }

    const certsForClassic = await chatCerts();
    try {
      await client.verifySignedJwtWithCertsAsync(token, certsForClassic, projectNumber, [CHAT_ISSUER]);
      return { ok: true };
    } catch (err) {
      return { ok: false, reason: `add-on: ${addOnReason}; classic: ${reasonOf(err)}` };
    }
  };
}

async function fetchChatCerts() {
  const res = await fetch(CHAT_CERTS_URL, { signal: AbortSignal.timeout(CERTS_FETCH_TIMEOUT_MS) });
  if (!res.ok) throw new Error(`Fetching Chat certs failed: HTTP ${res.status}`);
  return res.json();
}

module.exports = { createRequestVerifier };

import { createRemoteJWKSet, jwtVerify } from "jose";

const PER_PAGE = 1000;

// Reused across requests so jose can cache Cloudflare's signing keys.
let jwks;
let jwksTeamDomain;

export default {
  async fetch(request, env) {
    if (request.method !== "POST") {
      return json({ success: false, error: "method not allowed" }, 405);
    }

    // jose skips the audience check when POLICY_AUD is unset, which would accept
    // tokens issued for any Access application in the team.
    if (!env.TEAM_DOMAIN?.trim() || !env.POLICY_AUD?.trim()) {
      return json({ success: false, error: "TEAM_DOMAIN and POLICY_AUD must be set" }, 500);
    }

    // Reject anything that didn't come through Cloudflare Access.
    const token = request.headers.get("Cf-Access-Jwt-Assertion");
    if (!token) {
      return json({ success: false, error: "missing Access token" }, 403);
    }
    try {
      if (jwksTeamDomain !== env.TEAM_DOMAIN) {
        jwks = createRemoteJWKSet(new URL(`https://${env.TEAM_DOMAIN}/cdn-cgi/access/certs`));
        jwksTeamDomain = env.TEAM_DOMAIN;
      }
      await jwtVerify(token, jwks, {
        issuer: `https://${env.TEAM_DOMAIN}`,
        audience: env.POLICY_AUD,
      });
    } catch {
      return json({ success: false, error: "invalid Access token" }, 403);
    }

    let devices;
    try {
      const body = await request.json();
      devices = Array.isArray(body.devices) ? body.devices : [];
    } catch {
      return json({ success: false, error: "invalid request body" }, 400);
    }

    let hostsBySerial;
    try {
      hostsBySerial = await loadFleetHosts(env);
    } catch (err) {
      console.error(err);
      return json({ success: false, error: "failed to query Fleet" }, 502);
    }

    // Every device in the request must appear in the response.
    // Devices Fleet doesn't know about get a score of 0.
    const result = {};
    for (const device of devices) {
      const host = hostsBySerial.get(normalizeSerial(device.serial_number));
      result[device.device_id] = host
        ? { s2s_id: String(host.id), score: host.compliant ? 100 : 0 }
        : { s2s_id: "", score: 0 };
    }
    return json({ result });
  },
};

async function loadFleetHosts(env) {
  const hosts = await listHosts(env);

  // With POLICY_IDS set, only those policies count. Otherwise, any failing policy counts.
  const policyIds = (env.POLICY_IDS || "")
    .split(",")
    .map((id) => id.trim())
    .filter(Boolean);
  let failingHostIds = null;
  if (policyIds.length > 0) {
    failingHostIds = new Set();
    for (const policyId of policyIds) {
      const failing = await listHosts(env, { policy_id: policyId, policy_response: "failing" });
      for (const host of failing) failingHostIds.add(host.id);
    }
  }

  const bySerial = new Map();
  for (const host of hosts) {
    const serial = normalizeSerial(host.hardware_serial);
    if (!serial) continue;
    const failing = failingHostIds
      ? failingHostIds.has(host.id)
      : (host.issues?.failing_policies_count ?? 0) > 0;
    // If two hosts share a serial, both must pass.
    const previous = bySerial.get(serial);
    bySerial.set(serial, {
      id: host.id,
      compliant: !failing && (previous ? previous.compliant : true),
    });
  }
  return bySerial;
}

async function listHosts(env, filters = {}) {
  const hosts = [];
  // Page by host ID. Without order_key, Fleet doesn't sort hosts, so offset
  // pages can repeat or skip hosts.
  let lastId;
  for (;;) {
    const params = new URLSearchParams({ ...filters, order_key: "id", per_page: String(PER_PAGE) });
    if (lastId !== undefined) params.set("after", String(lastId));
    const res = await fetch(`${env.FLEET_URL}/api/v1/fleet/hosts?${params}`, {
      headers: {
        Authorization: `Bearer ${env.FLEET_API_TOKEN}`,
        Accept: "application/json",
      },
    });
    if (!res.ok) {
      throw new Error(`Fleet API returned ${res.status} for ${res.url}`);
    }
    const body = await res.json();
    hosts.push(...body.hosts);
    if (body.hosts.length < PER_PAGE) return hosts;
    lastId = body.hosts[body.hosts.length - 1].id;
  }
}

function normalizeSerial(serial) {
  return (serial || "").trim().toUpperCase();
}

function json(body, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

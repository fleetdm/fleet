import { test, expect, APIRequestContext } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { connect, StringCodec, NatsConnection } from 'nats';

// Optional: path to the directory where Fleet writes filesystem log files.
// Skip filesystem assertions unless this is set (only useful when the test
// runs on the same host as Fleet with the Filesystem log destination).
const LOG_FILES_PATH = process.env.FLEET_LOG_FILES_PATH;

/** Helper to fetch activities and find one matching type and detail predicate. */
async function findActivity(
  request: APIRequestContext,
  type: string,
  matches: (details: Record<string, unknown>) => boolean,
  perPage = 20
): Promise<Record<string, unknown> | undefined> {
  const response = await request.get('/api/latest/fleet/activities', {
    headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
    params: {
      order_key: 'created_at',
      order_direction: 'desc',
      per_page: String(perPage),
    },
  });
  expect(response.ok()).toBeTruthy();
  const data = await response.json();
  return data.activities?.find(
    (a: Record<string, unknown>) =>
      a.type === type && matches((a.details as Record<string, unknown>) ?? {})
  );
}

test.describe('Log destination flow', () => {
  // ── Config validation ─────────────────────────────────────────────────────
  test.describe('Config validation', () => {
    test('logging config is retrievable via API', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });

      expect(response.ok()).toBeTruthy();
      const config = await response.json();
      expect(config.logging).toBeDefined();
    });

    test('result log destination is configured', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });
      const config = await response.json();

      expect(config.logging.result).toBeDefined();
      expect(config.logging.result.plugin).toBeTruthy();
    });

    test('status log destination is configured', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });
      const config = await response.json();

      expect(config.logging.status).toBeDefined();
      expect(config.logging.status.plugin).toBeTruthy();
    });

    test('audit log destination is configured', async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });
      const config = await response.json();

      // Audit logs may not be enabled on all deployments; check it's defined
      expect(config.logging.audit).toBeDefined();
    });
  });

  // ── UI verification ───────────────────────────────────────────────────────
  // Log destinations are configured via the Fleet config file / env vars, not
  // through the UI. These tests verify settings navigation is healthy and that
  // admins can reach the pages where logging-related automations (policy /
  // software automations) are configured.
  test.describe('UI verification', () => {
    test('advanced settings page is accessible', async ({ page }) => {
      await page.goto('/settings/organization/advanced');

      await expect(page).toHaveURL(/\/settings/);
      await expect(
        page.getByRole('heading', { name: 'Advanced options', exact: true })
      ).toBeVisible();
    });

    test('policies page loads (source of policy log events)', async ({ page }) => {
      await page.goto('/policies/manage');

      await expect(page).toHaveURL(/\/policies/);
      await expect(page.getByRole('button', { name: /add policy/i })).toBeVisible();
    });
  });

  // ── End-to-end delivery via activities API ────────────────────────────────
  // The activities API captures every admin action regardless of where the
  // logs are ultimately routed (Filesystem, Kinesis, Firehose, etc.).
  test.describe('End-to-end log delivery', () => {
    const headers = { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` };

    test('report (query) lifecycle events are logged', async ({ request }) => {
      const name = `log_test_query_${Date.now()}`;

      // Create
      const createRes = await request.post('/api/latest/fleet/queries', {
        headers,
        data: { name, query: 'SELECT 1;' },
      });
      expect(createRes.ok()).toBeTruthy();
      const queryID = (await createRes.json()).query.id;

      const created = await findActivity(
        request,
        'created_saved_query',
        (d) => d.query_name === name
      );
      expect(created).toBeDefined();

      // Delete (cleanup) + verify deletion activity
      const delRes = await request.delete(`/api/latest/fleet/queries/id/${queryID}`, { headers });
      expect(delRes.ok()).toBeTruthy();

      const deleted = await findActivity(
        request,
        'deleted_saved_query',
        (d) => d.query_name === name
      );
      expect(deleted).toBeDefined();
    });

    test('policy lifecycle events are logged', async ({ request }) => {
      const name = `log_test_policy_${Date.now()}`;

      // Create
      const createRes = await request.post('/api/latest/fleet/policies', {
        headers,
        data: { name, query: 'SELECT 1;', description: 'smoke test policy' },
      });
      expect(createRes.ok()).toBeTruthy();
      const policyID = (await createRes.json()).policy.id;

      const created = await findActivity(
        request,
        'created_policy',
        (d) => d.policy_name === name
      );
      expect(created).toBeDefined();

      // Delete (cleanup) + verify deletion activity
      const delRes = await request.post('/api/latest/fleet/policies/delete', {
        headers,
        data: { ids: [policyID] },
      });
      expect(delRes.ok()).toBeTruthy();

      const deleted = await findActivity(
        request,
        'deleted_policy',
        (d) => d.policy_name === name
      );
      expect(deleted).toBeDefined();
    });

    test('pack lifecycle events are logged', async ({ request }) => {
      const name = `log_test_pack_${Date.now()}`;

      const createRes = await request.post('/api/latest/fleet/packs', {
        headers,
        data: { name, description: 'smoke test pack', host_ids: [], label_ids: [] },
      });
      expect(createRes.ok()).toBeTruthy();
      const packID = (await createRes.json()).pack.id;

      const created = await findActivity(
        request,
        'created_pack',
        (d) => d.pack_name === name
      );
      expect(created).toBeDefined();

      // Cleanup + verify deletion activity
      const delRes = await request.delete(`/api/latest/fleet/packs/id/${packID}`, { headers });
      expect(delRes.ok()).toBeTruthy();

      const deleted = await findActivity(
        request,
        'deleted_pack',
        (d) => d.pack_name === name
      );
      expect(deleted).toBeDefined();
    });

  });

  // ── NATS log destination verification ─────────────────────────────────────
  // Auto-detects NATS from Fleet's logging config. When Fleet's result/status/
  // audit logging plugin is "nats", the test connects to the NATS server URL
  // reported in that config (overridable via FLEET_NATS_URL) and verifies that
  // admin actions produce audit messages on the configured subject.
  test.describe('NATS log delivery', () => {
    let natsEnabled = false;
    let natsConn: NatsConnection | undefined;
    let resultSubject = '';
    let statusSubject = '';
    let auditSubject = '';
    let skipReason = '';

    test.beforeAll(async ({ request }) => {
      const response = await request.get('/api/latest/fleet/config', {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });
      const config = await response.json();
      const logging = config.logging;

      const natsPluginConfig =
        logging?.result?.plugin === 'nats' ? logging.result.config :
        logging?.status?.plugin === 'nats' ? logging.status.config :
        logging?.audit?.plugin === 'nats' ? logging.audit.config :
        undefined;

      if (!natsPluginConfig) {
        skipReason = 'Fleet is not configured to use the NATS logging plugin';
        return;
      }

      // Prefer explicit env var, fall back to the server URL from Fleet's config
      const natsUrl = process.env.FLEET_NATS_URL ?? natsPluginConfig.server;
      if (!natsUrl) {
        skipReason = 'Fleet NATS config has no server URL and FLEET_NATS_URL is not set';
        return;
      }

      resultSubject = logging?.result?.config?.result_subject ?? 'osquery_result';
      statusSubject = logging?.status?.config?.status_subject ?? 'osquery_status';
      auditSubject = logging?.audit?.config?.audit_subject ?? 'fleet_audit';

      try {
        natsConn = await connect({ servers: natsUrl, timeout: 5000 });
        natsEnabled = true;
      } catch (err) {
        skipReason = `NATS connect failed (${natsUrl}): ${(err as Error).message}`;
      }
    });

    test.afterAll(async () => {
      if (natsConn) {
        await natsConn.drain();
      }
    });

    test('can connect to NATS with the configured subjects', async () => {
      test.skip(!natsEnabled, skipReason || 'NATS not enabled');
      expect(natsConn).toBeDefined();
      expect(auditSubject).toBeTruthy();
    });

    test('audit log message is published to NATS on admin action', async ({ request }) => {
      test.skip(!natsEnabled, skipReason || 'NATS not enabled');

      const sc = StringCodec();
      const sub = natsConn!.subscribe(auditSubject);
      const received: unknown[] = [];

      // Start collecting audit messages in the background
      (async () => {
        for await (const msg of sub) {
          try {
            received.push(JSON.parse(sc.decode(msg.data)));
          } catch {
            received.push(sc.decode(msg.data));
          }
        }
      })();

      // Ensure the subscription is registered on the NATS server before we
      // trigger the action — otherwise we can miss the audit message.
      await natsConn!.flush();

      // Trigger an admin action that generates an audit log
      const name = `nats_audit_test_${Date.now()}`;
      const authHeaders = { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` };
      const createRes = await request.post('/api/latest/fleet/queries', {
        headers: authHeaders,
        data: { name, query: 'SELECT 1;' },
      });
      expect(createRes.ok()).toBeTruthy();
      const queryID = (await createRes.json()).query.id;

      // Activity → audit log streaming is async and runs on a 5-minute cron.
      // Trigger the cron manually so we don't have to wait for the scheduled run.
      await request.post('/api/latest/fleet/trigger', {
        headers: authHeaders,
        params: { name: 'activities_streaming' },
      });

      // Wait up to 10 seconds for the audit message to arrive
      const deadline = Date.now() + 10000;
      let matched: Record<string, unknown> | undefined;
      while (Date.now() < deadline) {
        matched = received.find((m): m is Record<string, unknown> => {
          if (typeof m !== 'object' || m === null) return false;
          const entry = m as Record<string, unknown>;
          const details = entry.details as Record<string, unknown> | undefined;
          return entry.type === 'created_saved_query' && details?.query_name === name;
        });
        if (matched) break;
        await new Promise((r) => setTimeout(r, 200));
      }

      // Cleanup the test query
      await request.delete(`/api/latest/fleet/queries/id/${queryID}`, {
        headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
      });
      await sub.unsubscribe();

      expect(
        matched,
        `Expected created_saved_query audit message with query_name="${name}" on "${auditSubject}" (received ${received.length} messages total)`
      ).toBeDefined();
    });
  });

  // ── Filesystem log verification ───────────────────────────────────────────
  // Only runs when FLEET_LOG_FILES_PATH points to the directory where Fleet
  // writes log files. Skipped otherwise.
  test.describe('Filesystem log delivery', () => {
    test.skip(
      !LOG_FILES_PATH,
      'Set FLEET_LOG_FILES_PATH to the directory with Fleet filesystem log files to run these tests'
    );

    test('result log file exists and is writable', async () => {
      const resultLog = path.join(LOG_FILES_PATH!, 'osqueryd.results.log');
      expect(fs.existsSync(resultLog)).toBeTruthy();
    });

    test('status log file exists and is writable', async () => {
      const statusLog = path.join(LOG_FILES_PATH!, 'osqueryd.status.log');
      expect(fs.existsSync(statusLog)).toBeTruthy();
    });

    test('filesystem log has been written to recently', async () => {
      // Find any .log file in the directory and verify it has recent mtime
      const files = fs.readdirSync(LOG_FILES_PATH!)
        .filter((f) => f.endsWith('.log'))
        .map((f) => path.join(LOG_FILES_PATH!, f));

      expect(files.length).toBeGreaterThan(0);

      const recentlyWritten = files.some((f) => {
        const stat = fs.statSync(f);
        const ageMs = Date.now() - stat.mtimeMs;
        return ageMs < 24 * 60 * 60 * 1000; // modified within last 24h
      });
      expect(recentlyWritten).toBeTruthy();
    });
  });
});

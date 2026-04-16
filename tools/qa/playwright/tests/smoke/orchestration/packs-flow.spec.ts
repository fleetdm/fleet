import { test, expect, APIRequestContext } from '@playwright/test';
import { tableRow } from '../../../helpers/nav';

const PACK_NAME = `Smoke Pack ${Date.now()}`;
const PACK_DESCRIPTION = 'Automated smoke test pack';

/** Fetch the most recent activities and find one matching the given type and pack name. */
async function findActivity(
  request: APIRequestContext,
  type: string,
  packName: string,
  perPage = 10
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
      a.type === type &&
      (a.details as Record<string, unknown>)?.pack_name === packName
  );
}

test.describe('Packs flow', () => {
  test.describe.configure({ mode: 'serial' });

  test('packs page is accessible', async ({ page }) => {
    await page.goto('/packs/manage');

    await expect(page).toHaveURL(/\/packs/);
    await expect(page.getByRole('heading', { name: 'Packs', exact: true })).toBeVisible();
  });

  test('can create a new pack with a host target', async ({ page, request }) => {
    await page.goto('/packs/manage');
    await page.getByRole('button', { name: /create new pack/i }).click();

    await expect(page).toHaveURL(/\/packs\/new/);

    // Fill in pack details
    await page.getByRole('textbox', { name: 'Name' }).fill(PACK_NAME);
    await page.getByRole('textbox', { name: 'Description' }).fill(PACK_DESCRIPTION);

    // Open the target selector and search for hosts
    await page.locator('.Select-placeholder').click();
    await page.locator('.Select-input input').fill('a');

    // Wait for the hosts section to appear in the dropdown
    await expect(page.locator('.Select-menu')).toBeVisible();

    // Click the "+" button on the first available host to add it as a target
    const hostOption = page.locator('.target-option__wrapper.is-host').first();
    await expect(hostOption).toBeVisible();
    await hostOption.locator('svg').first().click({ force: true });

    // Confirm the target was added
    await expect(page.getByText(/1 unique host/)).toBeVisible();

    // Save the pack
    await page.getByRole('button', { name: /save query pack/i }).click();

    // Should navigate to the edit page for the new pack
    await expect(page).toHaveURL(/\/packs\/\d+/);
    await expect(page.getByRole('heading', { name: /edit pack/i })).toBeVisible();

    // Verify the "created_pack" activity was logged
    const activity = await findActivity(request, 'created_pack', PACK_NAME);
    expect(activity).toBeDefined();
    expect(activity!.actor_email).toBe(process.env.FLEET_ADMIN_EMAIL);
  });

  test('pack appears in the packs list after creation', async ({ page }) => {
    await page.goto('/packs/manage');

    await expect(page.getByRole('link', { name: PACK_NAME })).toBeVisible();
  });

  test('pack is targeting a host', async ({ page }) => {
    await page.goto('/packs/manage');

    // Find the row with our pack name and verify it shows 1 host
    const packRow = page.getByRole('row').filter({ hasText: PACK_NAME });
    await expect(packRow).toBeVisible();

    // The "Hosts" column should show at least 1
    const hostsCell = packRow.getByRole('cell').nth(3);
    const hostsText = await hostsCell.innerText();
    expect(Number(hostsText)).toBeGreaterThanOrEqual(1);
  });

  test('can edit an existing pack', async ({ page, request }) => {
    await page.goto('/packs/manage');
    await page.getByRole('link', { name: PACK_NAME }).click();

    await expect(page).toHaveURL(/\/packs\/\d+\/edit/);

    const updatedDescription = 'Updated smoke test pack description';
    await page.getByRole('textbox', { name: 'Description' }).fill(updatedDescription);
    await page.getByRole('button', { name: /save/i }).click();

    // Reload and verify the update persisted
    await page.goto('/packs/manage');
    await page.getByRole('link', { name: PACK_NAME }).click();
    await expect(page.getByRole('textbox', { name: 'Description' })).toHaveValue(updatedDescription);

    // Verify the "edited_pack" activity was logged
    const activity = await findActivity(request, 'edited_pack', PACK_NAME);
    expect(activity).toBeDefined();
    expect(activity!.actor_email).toBe(process.env.FLEET_ADMIN_EMAIL);
  });

  test('can delete a pack', async ({ page, request }) => {
    await page.goto('/packs/manage');

    // Select the pack via its checkbox
    const packRow = page.getByRole('row').filter({ hasText: PACK_NAME });
    await packRow.getByRole('checkbox').click();

    // Click the delete action
    await page.getByRole('button', { name: /delete/i }).click();

    // Confirm the deletion modal
    await page.getByRole('button', { name: /delete/i }).last().click();

    await expect(page.getByText(PACK_NAME)).not.toBeVisible();

    // Verify the "deleted_pack" activity was logged
    const activity = await findActivity(request, 'deleted_pack', PACK_NAME);
    expect(activity).toBeDefined();
    expect(activity!.actor_email).toBe(process.env.FLEET_ADMIN_EMAIL);
  });

  test('pack query executes on targeted host', async ({ request }) => {
    // Increase timeout — osquery config refresh can take up to a few minutes
    test.setTimeout(5 * 60_000);

    const headers = { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` };

    // 1. Find the first available host
    const hostsRes = await request.get('/api/latest/fleet/hosts', { headers });
    expect(hostsRes.ok()).toBeTruthy();
    const hostsData = await hostsRes.json();
    const host = hostsData.hosts?.[0];
    expect(host).toBeDefined();
    const hostID = host.id;

    // 2. Find or create a simple query to schedule in the pack
    const queriesRes = await request.get('/api/latest/fleet/queries', { headers });
    expect(queriesRes.ok()).toBeTruthy();
    const queriesData = await queriesRes.json();
    let queryID: number;

    if (queriesData.queries?.length > 0) {
      queryID = queriesData.queries[0].id;
    } else {
      // Create a lightweight query
      const createQueryRes = await request.post('/api/latest/fleet/queries', {
        headers,
        data: { query: 'SELECT 1;', name: `smoke_pack_query_${Date.now()}` },
      });
      expect(createQueryRes.ok()).toBeTruthy();
      queryID = (await createQueryRes.json()).query.id;
    }

    // 3. Create a pack targeting the host
    const execPackName = `Exec Pack ${Date.now()}`;
    const createPackRes = await request.post('/api/latest/fleet/packs', {
      headers,
      data: {
        name: execPackName,
        description: 'Smoke test: verify pack query execution',
        host_ids: [hostID],
      },
    });
    expect(createPackRes.ok()).toBeTruthy();
    const packID = (await createPackRes.json()).pack.id;

    // 4. Schedule the query in the pack with a short interval
    const scheduleRes = await request.post('/api/latest/fleet/packs/schedule', {
      headers,
      data: { pack_id: packID, query_id: queryID, interval: 10 },
    });
    expect(scheduleRes.ok()).toBeTruthy();

    // 5. Poll host pack_stats until our pack appears with executions > 0.
    //    A freshly created pack will NOT appear in pack_stats until the host
    //    fetches the new config and reports back — so its mere presence with
    //    executions > 0 proves the query ran during THIS test, not a prior run.
    const timeoutMs = 4 * 60_000; // 4 minutes max
    const pollIntervalMs = 15_000; // check every 15 seconds
    let executed = false;

    const deadline = Date.now() + timeoutMs;
    while (Date.now() < deadline) {
      const hostRes = await request.get(`/api/latest/fleet/hosts/${hostID}`, { headers });
      expect(hostRes.ok()).toBeTruthy();
      const hostData = await hostRes.json();
      const packStats = hostData.host?.pack_stats as Array<{
        pack_id: number;
        pack_name: string;
        query_stats: Array<{ executions: number; last_executed: string }>;
      }> | undefined;

      const ourPack = packStats?.find((p) => p.pack_id === packID);
      if (ourPack?.query_stats?.some((q) => q.executions > 0)) {
        executed = true;
        break;
      }

      await new Promise((r) => setTimeout(r, pollIntervalMs));
    }

    // 6. Clean up — delete the pack regardless of result
    await request.delete(`/api/latest/fleet/packs/id/${packID}`, { headers });

    expect(executed).toBeTruthy();
  });

  test('packs page loads without errors after migration', async ({ page }) => {
    await page.goto('/packs/manage');

    await expect(page).toHaveURL(/\/packs/);
    await expect(page.getByRole('heading', { name: 'Packs', exact: true })).toBeVisible();
    await expect(
      tableRow(page).or(page.locator('.empty-table__container'))
    ).toBeVisible();
  });
});

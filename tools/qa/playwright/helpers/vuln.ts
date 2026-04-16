import { Page, Locator, expect } from '@playwright/test';
import { tableRow, tableOrEmpty } from './nav';

// ── Types ────────────────────────────────────────────────────────────────────

export interface SoftwareRef {
  name: string;
  type: string;
}

export interface HostRef {
  id: number;
  displayName: string;
}

// ── API utilities ────────────────────────────────────────────────────────────

/** Fetch a Fleet API token using admin credentials from env, or use FLEET_API_TOKEN if valid. */
export async function getApiToken(baseURL: string): Promise<string> {
  if (process.env.FLEET_API_TOKEN) {
    const check = await fetch(`${baseURL}/api/v1/fleet/me`, {
      headers: { Authorization: `Bearer ${process.env.FLEET_API_TOKEN}` },
    });
    if (check.ok) return process.env.FLEET_API_TOKEN;
  }

  const res = await fetch(`${baseURL}/api/v1/fleet/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      email: process.env.FLEET_ADMIN_EMAIL,
      password: process.env.FLEET_ADMIN_PASSWORD,
    }),
  });
  if (!res.ok) throw new Error(`Login failed: ${res.status}`);
  return (await res.json()).token;
}

/**
 * Match a host's actual platform field against the desired OS group.
 * The Fleet API `platform` query param returns a label group which may include
 * hosts from other platforms, so we filter client-side by the real platform value.
 */
function matchesPlatform(
  hostPlatform: string,
  desired: 'darwin' | 'windows' | 'linux',
): boolean {
  if (desired === 'darwin') return hostPlatform === 'darwin';
  if (desired === 'windows') return hostPlatform === 'windows';
  const linuxPlatforms = [
    'linux', 'ubuntu', 'debian', 'rhel', 'centos', 'arch',
    'fedora', 'amzn', 'sles', 'gentoo', 'pop', 'manjaro',
  ];
  return linuxPlatforms.includes(hostPlatform);
}

/** Find a host of a given platform that has vulnerable software. */
export async function findHostByPlatform(
  baseURL: string,
  token: string,
  platform: 'darwin' | 'windows' | 'linux',
): Promise<HostRef | null> {
  const res = await fetch(
    `${baseURL}/api/v1/fleet/hosts?platform=${platform}&per_page=50`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  if (!res.ok) return null;
  const { hosts } = await res.json();
  if (!hosts?.length) return null;

  const filtered = hosts.filter((h: { platform: string }) =>
    matchesPlatform(h.platform, platform),
  );

  for (const host of filtered) {
    const swRes = await fetch(
      `${baseURL}/api/v1/fleet/hosts/${host.id}/software?vulnerable=true&per_page=1`,
      { headers: { Authorization: `Bearer ${token}` } },
    );
    if (!swRes.ok) continue;
    const swBody = await swRes.json();
    if (swBody.software?.length > 0 || swBody.count > 0) {
      return { id: host.id, displayName: host.display_name };
    }
  }
  return null;
}

// ── Table cell locators ──────────────────────────────────────────────────────
// Locate cells by their visible column header text — no class names, no hardcoded indices.

/**
 * Resolve the column index for a given header name.
 * Throws if the column is not found.
 */
async function resolveColumnIndex(page: Page, headerName: string): Promise<number> {
  const headers = page.getByRole('table').locator('thead th');
  const count = await headers.count();
  for (let i = 0; i < count; i++) {
    const text = (await headers.nth(i).innerText()).trim();
    if (text === headerName) return i;
  }
  throw new Error(`Column "${headerName}" not found in table headers`);
}

/**
 * Get the cell in a row that corresponds to a visible column header.
 * Works regardless of column order.
 */
export async function cellByColumn(
  page: Page,
  row: Locator,
  headerName: string,
): Promise<Locator> {
  const idx = await resolveColumnIndex(page, headerName);
  return row.locator('td').nth(idx);
}

// ── UI interaction helpers ───────────────────────────────────────────────────

/**
 * Open the filters modal, enable "Vulnerable software", and apply.
 * Works on both /software/titles and host details > Software tab — same modal.
 */
export async function applyVulnerableFilter(page: Page): Promise<void> {
  await page.getByRole('button', { name: /filter/i }).click();
  await page.locator('form').getByRole('switch').click();

  // Capture item count before applying so we can detect when data re-renders
  const countBefore = await page
    .locator('text=/\\d[\\d,]*\\s+items?/')
    .first()
    .innerText()
    .catch(() => '');

  await page.getByRole('button', { name: 'Apply' }).click();

  // Wait for the filtered data to fully render: item count must change
  if (countBefore) {
    await expect(
      page.locator('text=/\\d[\\d,]*\\s+items?/').first(),
    ).not.toHaveText(countBefore, { timeout: 10000 });
  }
  await expect(tableOrEmpty(page)).toBeVisible();
}

/**
 * Wait for a table page transition to complete after clicking Next/Previous
 * or switching tabs. The DataTable renders a .loading-overlay during fetches.
 */
export async function waitForTableReload(page: Page): Promise<void> {
  const overlay = page.locator('.loading-overlay');
  // The overlay may appear briefly — wait for it to attach then detach.
  // If the response is cached it may never appear, so use a short timeout for attach.
  await overlay.waitFor({ state: 'attached', timeout: 2000 }).catch(() => {});
  await overlay.waitFor({ state: 'detached', timeout: 10000 });
  await expect(tableRow(page)).toBeVisible();
}

// ── Assertion helpers ────────────────────────────────────────────────────────

/**
 * Assert that a row's "Vulnerabilities" cell shows data (not "---").
 */
export async function expectRowHasVulnData(
  page: Page,
  row: Locator,
): Promise<void> {
  const cell = await cellByColumn(page, row, 'Vulnerabilities');
  await expect(cell).not.toHaveText('---');
}

/**
 * Assert that a row's "Vulnerabilities" cell shows a single CVE identifier.
 * Returns the CVE text.
 */
export async function expectSingleCve(
  page: Page,
  row: Locator,
): Promise<string> {
  const cell = await cellByColumn(page, row, 'Vulnerabilities');
  await expect(cell).toHaveText(/^CVE-\d{4}-\d+$/);
  return (await cell.innerText()).trim();
}

/**
 * Hover over a multi-vulnerability cell and assert the tooltip shows CVE entries.
 */
export async function assertVulnTooltip(
  page: Page,
  row: Locator,
): Promise<void> {
  const cell = await cellByColumn(page, row, 'Vulnerabilities');
  await cell.hover();

  const tooltip = page
    .getByRole('list')
    .filter({ has: page.getByRole('listitem').filter({ hasText: /^CVE-/ }) });
  await expect(tooltip).toBeVisible({ timeout: 5000 });
  await expect(tooltip.getByRole('listitem').first()).toHaveText(/^CVE-\d{4}-\d+$/);
}

/**
 * Assert all expected elements on a CVE detail page.
 * Set `clickNvdLink` to true to verify the NVD link opens in a new tab.
 */
export async function assertCveDetailPage(
  page: Page,
  expectedCve: string,
  { clickNvdLink = false } = {},
): Promise<void> {
  await page.waitForLoadState('networkidle');

  // CVE number in heading
  await expect(
    page.getByRole('heading', { name: expectedCve, level: 1 }),
  ).toBeVisible({ timeout: 10000 });

  // Metadata: at least "Detected" and "Affected hosts" labels are always present
  await expect(page.getByText('Detected')).toBeVisible();
  await expect(page.getByText('Affected hosts')).toBeVisible();

  // "Visit NVD page" link with correct href
  const nvdLink = page.getByRole('link', { name: 'Visit NVD page' });
  await expect(nvdLink).toBeVisible();
  await expect(nvdLink).toHaveAttribute(
    'href',
    `https://nvd.nist.gov/vuln/detail/${expectedCve}`,
  );

  if (clickNvdLink) {
    const [newPage] = await Promise.all([
      page.context().waitForEvent('page'),
      nvdLink.click(),
    ]);
    await newPage.waitForLoadState('domcontentloaded');
    expect(newPage.url()).toContain(
      `https://nvd.nist.gov/vuln/detail/${expectedCve}`,
    );
    await newPage.close();
  }

  // Vulnerable software table has at least one row
  await expect(tableRow(page)).toBeVisible();
}

// ── Discovery helpers ────────────────────────────────────────────────────────

/**
 * Scan paginated vulnerable software table to find one software per target type.
 * Matches the "Type" column against the target strings.
 */
export async function findSoftwareByTypes(
  page: Page,
  targetTypes: string[],
  maxPages = 10,
): Promise<Map<string, SoftwareRef>> {
  const found = new Map<string, SoftwareRef>();
  const remaining = new Set(targetTypes);

  for (let pageNum = 0; pageNum < maxPages && remaining.size > 0; pageNum++) {
    if (pageNum > 0) {
      const nextBtn = page.getByRole('button', { name: 'Next' });
      if (await nextBtn.isDisabled()) break;
      await nextBtn.click();
      await waitForTableReload(page);
    }

    const rows = page.getByRole('table').locator('tbody tr');
    const rowCount = await rows.count();

    for (let i = 0; i < rowCount && remaining.size > 0; i++) {
      const row = rows.nth(i);
      const typeCell = await cellByColumn(page, row, 'Type');
      const type = (await typeCell.innerText()).trim();

      for (const target of remaining) {
        if (type.includes(target)) {
          const nameCell = await cellByColumn(page, row, 'Name');
          const link = nameCell.getByRole('link').first();
          const name = (await link.innerText()).trim();
          found.set(target, { name, type });
          remaining.delete(target);
          break;
        }
      }
    }
  }

  return found;
}

/**
 * Find the first table row whose "Vulnerabilities" cell matches a pattern.
 * Returns the row locator, or null if none found on the current page.
 */
export async function findRowWithVulnPattern(
  page: Page,
  pattern: RegExp,
): Promise<Locator | null> {
  const rows = page.getByRole('table').locator('tbody tr');
  const count = await rows.count();
  for (let i = 0; i < count; i++) {
    const row = rows.nth(i);
    const cell = await cellByColumn(page, row, 'Vulnerabilities');
    const text = (await cell.innerText()).trim();
    if (pattern.test(text)) return row;
  }
  return null;
}

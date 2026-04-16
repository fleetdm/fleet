import { test, expect } from '@playwright/test';
import { tableRow } from '../../../helpers/nav';
import {
  type SoftwareRef,
  type HostRef,
  getApiToken,
  findHostByPlatform,
  applyVulnerableFilter,
  expectRowHasVulnData,
  expectSingleCve,
  assertVulnTooltip,
  assertCveDetailPage,
  cellByColumn,
  findSoftwareByTypes,
  findRowWithVulnPattern,
  waitForTableReload,
} from '../../../helpers/vuln';

test.describe.configure({ mode: 'serial' });

// ── Shared state ─────────────────────────────────────────────────────────────

const OS_KEYS = ['macos', 'deb', 'windows'] as const;
type OsKey = typeof OS_KEYS[number];

const TARGET_TYPES: Record<OsKey, string> = {
  macos: 'Application (macOS)',
  deb: 'Package (deb)',
  windows: 'Program (Windows)',
};

const OS_LABELS: Record<OsKey, string> = {
  macos: 'macOS',
  deb: 'Linux (deb)',
  windows: 'Windows',
};

let softwareByOS: Partial<Record<OsKey, SoftwareRef>> = {};
let hostByOS: Partial<Record<OsKey, HostRef>> = {};
let nvdLinkVerified = false;

// ── Setup: discover hosts via API ────────────────────────────────────────────

test.beforeAll(async () => {
  const baseURL = process.env.FLEET_URL!;
  const token = await getApiToken(baseURL);

  const [macHost, linuxHost, winHost] = await Promise.all([
    findHostByPlatform(baseURL, token, 'darwin'),
    findHostByPlatform(baseURL, token, 'linux'),
    findHostByPlatform(baseURL, token, 'windows'),
  ]);

  if (macHost) hostByOS.macos = macHost;
  if (linuxHost) hostByOS.deb = linuxHost;
  if (winHost) hostByOS.windows = winHost;
});

// ═════════════════════════════════════════════════════════════════════════════
// Test 1: Software Titles — filter, column assertions, and discovery
// ═════════════════════════════════════════════════════════════════════════════

test('Software Titles — vulnerable filter, pagination, and column checks', async ({
  page,
}) => {
  await page.goto('/software/titles');
  await expect(tableRow(page)).toBeVisible();

  // Apply vulnerable filter via UI
  await applyVulnerableFilter(page);
  await expect(page).toHaveURL(/vulnerable=true/);

  // Every row should show vulnerability data
  const rows = page.getByRole('table').locator('tbody tr');
  const rowCount = await rows.count();
  for (let i = 0; i < rowCount; i++) {
    await expectRowHasVulnData(page, rows.nth(i));
  }

  // Multi-vulnerability tooltip
  const multiRow = await findRowWithVulnPattern(page, /^\d+ vulnerabilities$/);
  if (multiRow) {
    await assertVulnTooltip(page, multiRow);
  }

  // Single vulnerability — CVE displayed directly
  const singleRow = await findRowWithVulnPattern(page, /^CVE-\d{4}-\d+$/);
  if (singleRow) {
    await expectSingleCve(page, singleRow);
  }

  // Pagination
  const nextBtn = page.getByRole('button', { name: 'Next' });
  if (!(await nextBtn.isDisabled())) {
    await nextBtn.click();
    await waitForTableReload(page);

    if (!(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await waitForTableReload(page);
    }

    await page.getByRole('button', { name: 'Previous' }).click();
    await waitForTableReload(page);
  }

  // Reset to page 1 via tab click, re-apply filter
  await page.getByRole('tab', { name: 'Software' }).click();
  await waitForTableReload(page);
  await applyVulnerableFilter(page);
  await expect(page).toHaveURL(/vulnerable=true/);

  // Discover one software per OS type for subsequent flow tests
  const found = await findSoftwareByTypes(
    page,
    Object.values(TARGET_TYPES),
    15,
  );
  for (const key of OS_KEYS) {
    const ref = found.get(TARGET_TYPES[key]);
    if (ref) softwareByOS[key] = ref;
  }

  expect(
    OS_KEYS.filter((k) => softwareByOS[k]).length,
    'Expected at least one OS type with vulnerable software',
  ).toBeGreaterThan(0);
});

// ═════════════════════════════════════════════════════════════════════════════
// Tests 2/3/4: Per-OS click-through flow — titles → version → CVE
// ═════════════════════════════════════════════════════════════════════════════

for (const osKey of OS_KEYS) {
  test(`${OS_LABELS[osKey]} — software titles → version → CVE detail flow`, async ({
    page,
  }) => {
    test.skip(!softwareByOS[osKey], `No ${OS_LABELS[osKey]} software found`);
    const ref = softwareByOS[osKey]!;

    // ── Start at software titles, filter, search ───────────────────────────
    await page.goto('/software/titles');
    await expect(tableRow(page)).toBeVisible();
    await applyVulnerableFilter(page);
    await expect(page).toHaveURL(/vulnerable=true/);

    await page.getByRole('textbox', { name: /Search/ }).fill(ref.name);
    await waitForTableReload(page);

    // Click into the software title
    const firstRow = page.getByRole('table').locator('tbody tr').first();
    const nameLink = firstRow.locator('td').first().getByRole('link').first();
    await expect(nameLink).toContainText(ref.name);
    await nameLink.click();

    // ── Software title detail page ─────────────────────────────────────────
    await expect(tableRow(page)).toBeVisible();

    // Find a version with vulnerabilities and click it
    const versionRows = page.getByRole('table').locator('tbody tr');
    const versionCount = await versionRows.count();
    let clickedVersion = false;

    for (let i = 0; i < versionCount; i++) {
      const row = versionRows.nth(i);
      const vulnCell = await cellByColumn(page, row, 'Vulnerabilities');
      const vulnText = (await vulnCell.innerText()).trim();
      if (vulnText !== '---') {
        await expectRowHasVulnData(page, row);
        // Click the version link (first cell)
        await row.locator('td').first().getByRole('link').first().click();
        clickedVersion = true;
        break;
      }
    }
    expect(clickedVersion, 'Should find a version with vulnerabilities').toBe(true);

    // ── Software version detail page ───────────────────────────────────────
    await expect(tableRow(page)).toBeVisible();

    const cveRow = page.getByRole('table').locator('tbody tr').first();
    const cveCell = await cellByColumn(page, cveRow, 'Vulnerability');
    const cveLink = cveCell.getByRole('link');
    await expect(cveLink).toHaveText(/^CVE-\d{4}-\d+$/);
    const cveText = (await cveLink.innerText()).trim();
    await cveLink.click();

    // ── CVE detail page ────────────────────────────────────────────────────
    await expect(page).toHaveURL(/\/software\/vulnerabilities\/CVE-/);
    const shouldClickNvd = !nvdLinkVerified;
    await assertCveDetailPage(page, cveText, { clickNvdLink: shouldClickNvd });
    if (shouldClickNvd) nvdLinkVerified = true;
  });
}

// ═════════════════════════════════════════════════════════════════════════════
// Test 5: Vulnerabilities tab — navigate via tab click, list → CVE detail
// ═════════════════════════════════════════════════════════════════════════════

test('Vulnerabilities tab — list, pagination, and CVE detail flow', async ({
  page,
}) => {
  await page.goto('/software/titles');
  await expect(tableRow(page)).toBeVisible();

  // Navigate to vulnerabilities via tab
  await page.getByRole('tab', { name: 'Vulnerabilities' }).click();
  await expect(page).toHaveURL(/\/software\/vulnerabilities/);
  await expect(tableRow(page)).toBeVisible();

  // Read first CVE before paginating
  const firstRow = page.getByRole('table').locator('tbody tr').first();
  const firstCveCell = await cellByColumn(page, firstRow, 'Vulnerability');
  const firstCveLink = firstCveCell.getByRole('link');
  await expect(firstCveLink).toHaveText(/^CVE-\d{4}-\d+$/);
  const cveName = (await firstCveLink.innerText()).trim();

  // Paginate
  const nextBtn = page.getByRole('button', { name: 'Next' });
  if (!(await nextBtn.isDisabled())) {
    await nextBtn.click();
    await waitForTableReload(page);

    if (!(await nextBtn.isDisabled())) {
      await nextBtn.click();
      await waitForTableReload(page);
    }
  }

  // Return to page 1 via tab re-click
  await page.getByRole('tab', { name: 'Vulnerabilities' }).click();
  await waitForTableReload(page);

  // Click first CVE
  const cveRow = page.getByRole('table').locator('tbody tr').first();
  const cveCell = await cellByColumn(page, cveRow, 'Vulnerability');
  const cveLink = cveCell.getByRole('link');
  await expect(cveLink).toHaveText(cveName);
  await cveLink.click();

  await expect(page).toHaveURL(/\/software\/vulnerabilities\/CVE-/);
  await assertCveDetailPage(page, cveName);
});

// ═════════════════════════════════════════════════════════════════════════════
// Tests 6/7/8: Host Details — vulnerable software flow (per OS)
// ═════════════════════════════════════════════════════════════════════════════

for (const osKey of OS_KEYS) {
  test(`${OS_LABELS[osKey]} host — vulnerable software → version → CVE flow`, async ({
    page,
  }) => {
    test.skip(!hostByOS[osKey], `No ${OS_LABELS[osKey]} host with vulnerable software`);
    const host = hostByOS[osKey]!;

    // Navigate to host details > Software tab
    await page.goto(`/hosts/${host.id}`);
    await page.getByRole('tab', { name: 'Software' }).click();
    await expect(tableRow(page)).toBeVisible();

    // Apply vulnerable filter
    await applyVulnerableFilter(page);
    const hasRows = await tableRow(page).isVisible().catch(() => false);
    test.skip(!hasRows, `No vulnerable software on ${OS_LABELS[osKey]} host`);

    // Click first software
    const firstRow = page.getByRole('table').locator('tbody tr').first();
    await firstRow.locator('td').first().getByRole('link').first().click();

    // ── Software title detail page ─────────────────────────────────────────
    await expect(tableRow(page)).toBeVisible();

    const versionRows = page.getByRole('table').locator('tbody tr');
    const rowCount = await versionRows.count();
    let clickedVersion = false;

    for (let i = 0; i < rowCount; i++) {
      const row = versionRows.nth(i);
      const vulnCell = await cellByColumn(page, row, 'Vulnerabilities');
      const vulnText = (await vulnCell.innerText()).trim();
      if (vulnText !== '---') {
        await row.locator('td').first().getByRole('link').first().click();
        clickedVersion = true;

        // ── Software version detail page ───────────────────────────────────
        await expect(tableRow(page)).toBeVisible();

        const cveRow = page.getByRole('table').locator('tbody tr').first();
        const cveCellLoc = await cellByColumn(page, cveRow, 'Vulnerability');
        const cveLink = cveCellLoc.getByRole('link');
        await expect(cveLink).toHaveText(/^CVE-\d{4}-\d+$/);
        const cveText = (await cveLink.innerText()).trim();
        await cveLink.click();

        // ── CVE detail page ────────────────────────────────────────────────
        await expect(page).toHaveURL(/\/software\/vulnerabilities\/CVE-/);
        await assertCveDetailPage(page, cveText);
        break;
      }
    }
    expect(clickedVersion, 'Should find a version with vulnerabilities').toBe(true);
  });
}

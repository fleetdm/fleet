import { Page, Locator, expect } from '@playwright/test';

// ── Table locators ────────────────────────────────────────────────────────────

/** First data row in a table */
export function tableRow(page: Page): Locator {
  return page.getByRole('table').locator('tbody').getByRole('row').first();
}

/** First data row that contains a link — confirms actual content has rendered, not just the table shell. */
export function tableRowWithContent(page: Page): Locator {
  return page.getByRole('table').locator('tbody').getByRole('row').filter({ has: page.getByRole('link') }).first();
}

/** First data row OR empty state container */
export function tableOrEmpty(page: Page): Locator {
  return tableRow(page).or(page.locator('.empty-table__container'));
}

/**
 * First match of either a populated element or an empty state.
 * Accepts comma-separated selectors for either side.
 */
export function contentOrEmpty(page: Page, populated: string, empty: string): Locator {
  return page.locator(populated).or(page.locator(empty)).first();
}

// ── Cell text extraction ──────────────────────────────────────────────────────

/**
 * Get visible text from a table cell, avoiding hidden tooltip/badge content.
 * Skips the first column if it's a checkbox (index 0).
 */
export async function getNameFromRow(page: Page, cellIndex = 1): Promise<string> {
  const cell = tableRow(page).locator('td').nth(cellIndex);
  // Try the truncated text span first (avoids badges like "Inherited")
  const truncatedText = cell.locator('.data-table__tooltip-truncated-text');
  if (await truncatedText.count() > 0) {
    return (await truncatedText.innerText()).trim();
  }
  return (await cell.innerText()).trim();
}

// ── Dropdown interactions ─────────────────────────────────────────────────────

/** Select a team from the team dropdown. index=1 picks the first real team (0 is "All teams"). */
export async function selectTeam(page: Page, index = 1): Promise<void> {
  await page.locator('.team-dropdown__control').click();
  await page.locator('.team-dropdown__option').nth(index).click();
  await page.waitForURL(/fleet_id/);
}

/** Open the hosts status filter dropdown and select an option by visible text. */
export async function selectStatusFilter(page: Page, status: string): Promise<void> {
  await page.locator('.manage-hosts__status-filter .react-select__control').click();
  const option = page.locator('[data-testid="dropdown-option"]').filter({ hasText: status });
  await expect(option).toBeVisible();
  await option.click();
}

/** Open the hosts label/platform filter dropdown and select by text. */
export async function selectPlatformFilter(page: Page, platform: string): Promise<void> {
  await page.locator('.label-filter-select__control').click();
  const option = page.locator('.label-filter-select__option').filter({ hasText: platform });
  await expect(option).toBeVisible();
  await option.click();
}

/**
 * Open the hosts label filter dropdown and select the first custom label
 * (skipping built-in platform labels).
 */
export async function selectFirstCustomLabel(page: Page): Promise<void> {
  const platforms = ['macOS', 'Windows', 'Linux', 'ChromeOS', 'iOS', 'iPadOS', 'Android'];
  await page.locator('.label-filter-select__control').click();
  const allOptions = page.locator('.label-filter-select__option');
  const count = await allOptions.count();
  for (let i = 0; i < count; i++) {
    const text = await allOptions.nth(i).textContent();
    if (text && !platforms.some((p) => text.includes(p))) {
      await allOptions.nth(i).click();
      return;
    }
  }
  // Fallback: click first option if no custom labels found
  await allOptions.first().click();
}

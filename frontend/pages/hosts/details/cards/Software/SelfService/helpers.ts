import {
  HOST_SOFTWARE_UI_IN_PROGRESS_STATUSES,
  HOST_SOFTWARE_UI_PENDING_STATUSES,
  IDeviceSoftwareWithUiStatus,
  SoftwareCategory,
} from "interfaces/software";
import { ISelfServiceCategory } from "interfaces/self_service_category";

type CategoryFilterValue = SoftwareCategory | "All";

/** Statuses indicating an item is in-progress (server-acknowledged but not finished). */
const IN_PROGRESS_UI_STATUSES = new Set<string>([
  ...HOST_SOFTWARE_UI_IN_PROGRESS_STATUSES,
  ...HOST_SOFTWARE_UI_PENDING_STATUSES,
]);

/** Statuses indicating the user cannot click "Install" — already done or in-flight. */
const INSTALLED_OR_IN_FLIGHT_UI_STATUSES = new Set<string>([
  ...IN_PROGRESS_UI_STATUSES,
  "installed",
  "recently_installed",
  "recently_updated",
  "update_available", // user clicks "Update", not "Install" — not eligible for install_all
  // `recently_uninstalled` = the user JUST uninstalled this; inventory hasn't
  // refreshed yet. Including it in install_all would immediately re-install
  // what they just removed. Once inventory catches up the status becomes
  // `uninstalled` and install_all will pick it up again — this is a transient
  // guard, not a permanent exclusion.
  "recently_uninstalled",
  // failed_uninstall variants: the item is still installed (the uninstall
  // failed). Install_all is for things the user doesn't have yet.
  "failed_uninstall",
  "failed_uninstall_installed",
  "failed_uninstall_update_available",
  // Script packages: `ran_script` means it already executed — don't re-run on
  // install_all. `never_ran_script` IS eligible (not in this set) so first-time
  // script runs are queued.
  "ran_script",
]);

export interface ICategory {
  /** Temporary Clientside IDs */
  id: number;
  /** Text shown in the UI */
  label: string;
  /** Text stored in the API */
  value: CategoryFilterValue;
}

const ALL_ITEM: ICategory = { id: 0, label: "All", value: "All" };

/** @deprecated Static fallback list — used by `SoftwareOptionsSelector` and
 * `SelfServicePreview` on the Software page. New code should consume the
 * dynamic `/software/self_service_categories` endpoint instead. */
export const CATEGORIES_ITEMS: ICategory[] = [
  { id: 1, label: "🌎 Browsers", value: "Browsers" },
  { id: 2, label: "👬 Communication", value: "Communication" },
  { id: 3, label: "🧰 Developer tools", value: "Developer tools" },
  { id: 4, label: "🖥️ Productivity", value: "Productivity" },
  { id: 5, label: "🔐 Security", value: "Security" },
  { id: 6, label: "🛠️ Utilities", value: "Utilities" },
];

/** @deprecated See `CATEGORIES_ITEMS`. */
export const CATEGORIES_NAV_ITEMS: ICategory[] = [
  ALL_ITEM,
  ...CATEGORIES_ITEMS,
];

/**
 * Strips a leading emoji + whitespace from a custom category name so it can be
 * compared against software's existing `categories: SoftwareCategory[]` enum.
 *
 * BE will eventually associate software to custom categories by ID and this
 * helper will become obsolete — the device software endpoint will accept
 * `category_id` and filter server-side. Until then this gives a best-effort
 * client-side fallback for dev mode (#46369).
 */
const stripEmojiPrefix = (name: string): string =>
  name.replace(/^[^\p{L}\p{N}]+/u, "").trim();

/**
 * Returns software in the given custom category. Best-effort name match — see
 * `stripEmojiPrefix` doc comment. Returns the unmodified list when category is
 * undefined (the "All" filter).
 */
export const filterSoftwareByCustomCategory = (
  software: IDeviceSoftwareWithUiStatus[],
  categories: ISelfServiceCategory[],
  categoryId?: number
): IDeviceSoftwareWithUiStatus[] => {
  if (categoryId === undefined) {
    return software;
  }
  const category = categories.find((c) => c.id === categoryId);
  // Categories may still be loading, or the URL may reference a deleted category.
  // Either way, falling through to `software` would surface the wrong "Install all" count
  // and let the user bulk-install items outside the requested category.
  if (!category) {
    return [];
  }
  const normalized = stripEmojiPrefix(category.name).toLowerCase();
  return software.filter((item) => {
    const itemCategories = [
      ...(item.software_package?.categories ?? []),
      ...(item.app_store_app?.categories ?? []),
    ];
    return itemCategories.some((c) => c.toLowerCase() === normalized);
  });
};

/** Count of items in the list that are eligible to be queued by install_all. */
export const countUninstalledForInstallAll = (
  software: IDeviceSoftwareWithUiStatus[]
): number =>
  software.filter(
    (item) => !INSTALLED_OR_IN_FLIGHT_UI_STATUSES.has(item.ui_status)
  ).length;

/** True if any item in the list is currently in-progress (install_all should
 * be disabled until they all leave that state). */
export const hasInProgressInstallAllItems = (
  software: IDeviceSoftwareWithUiStatus[]
): boolean =>
  software.some((item) => IN_PROGRESS_UI_STATUSES.has(item.ui_status));

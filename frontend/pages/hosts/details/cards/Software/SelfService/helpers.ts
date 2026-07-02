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

/** Statuses that specifically indicate an install_all-style operation is in
 * flight. Narrower than IN_PROGRESS_UI_STATUSES: updates and uninstalls aren't
 * triggered by install_all, so they shouldn't keep the button visible after
 * `uninstalledCount` drops to 0. */
const INSTALL_ALL_IN_FLIGHT_UI_STATUSES = new Set<string>([
  "installing",
  "running_script",
  "pending_install",
  "pending_script",
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

/** @deprecated Static fallback list — used by `SoftwareOptionsSelector` on the
 * Software page. New code should consume the dynamic
 * `/software/self_service_categories` endpoint instead. */
export const CATEGORIES_ITEMS: ICategory[] = [
  { id: 1, label: "🌎 Browsers", value: "Browsers" },
  { id: 2, label: "👬 Communication", value: "Communication" },
  { id: 3, label: "🧰 Developer tools", value: "Developer tools" },
  { id: 4, label: "🖥️ Productivity", value: "Productivity" },
  { id: 5, label: "🔐 Security", value: "Security" },
  { id: 6, label: "🛟 Support", value: "Support" },
  { id: 7, label: "🛠️ Utilities", value: "Utilities" },
];

// Client-side category filter by name — both sides come from
// `software_categories` until BE supports server-side `category_id` (#46369).
// `categoryId === undefined` is the "All" filter (returns input unchanged);
// an unknown id (stale URL or still-loading list) returns `[]`.
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
  const normalized = category.name.toLowerCase();
  return software.filter((item) => {
    const itemCategories = [
      ...(item.software_package?.categories ?? []),
      ...(item.app_store_app?.categories ?? []),
    ];
    return itemCategories.some((c) => c.toLowerCase() === normalized);
  });
};

// Returns only the categories that have at least one self-service software item
// assigned for this host, so empty categories are hidden from the filter
// (#48614). Category membership is resolved the same way as
// `filterSoftwareByCustomCategory` (matching on lowercased name across
// `software_package` and `app_store_app` categories) so the dropdown stays
// consistent with what selecting a category would actually show. The host's
// full self-service list is available client-side (the API isn't paginated), so
// this reflects MDM enrollment, label scoping, and platform exactly as resolved
// by the backend software query.
export const filterCategoriesWithSoftware = (
  categories: ISelfServiceCategory[],
  software: IDeviceSoftwareWithUiStatus[]
): ISelfServiceCategory[] => {
  const categoryNamesInUse = new Set<string>();
  software.forEach((item) => {
    [
      ...(item.software_package?.categories ?? []),
      ...(item.app_store_app?.categories ?? []),
    ].forEach((name) => categoryNamesInUse.add(name.toLowerCase()));
  });
  return categories.filter((c) => categoryNamesInUse.has(c.name.toLowerCase()));
};

/** Count of items in the list that are eligible to be queued by install_all. */
export const countUninstalledForInstallAll = (
  software: IDeviceSoftwareWithUiStatus[]
): number =>
  software.filter(
    (item) => !INSTALLED_OR_IN_FLIGHT_UI_STATUSES.has(item.ui_status)
  ).length;

/** True if any item in the list is currently being installed/scripted by an
 * install_all-style operation. Updates and uninstalls don't count — those are
 * orthogonal operations and shouldn't keep the install_all button visible. */
export const hasInProgressInstallAllItems = (
  software: IDeviceSoftwareWithUiStatus[]
): boolean =>
  software.some((item) =>
    INSTALL_ALL_IN_FLIGHT_UI_STATUSES.has(item.ui_status)
  );

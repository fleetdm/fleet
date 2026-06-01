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

export const CATEGORIES_ITEMS: ICategory[] = [
  { id: 1, label: "🌎 Browsers", value: "Browsers" },
  { id: 2, label: "👬 Communication", value: "Communication" },
  { id: 3, label: "🧰 Developer tools", value: "Developer tools" },
  { id: 4, label: "🖥️ Productivity", value: "Productivity" },
  { id: 5, label: "🔐 Security", value: "Security" },
  { id: 6, label: "🛠️ Utilities", value: "Utilities" },
];

export const CATEGORIES_NAV_ITEMS: ICategory[] = [
  ALL_ITEM,
  ...CATEGORIES_ITEMS,
];

export const filterSoftwareByCategory = (
  software?: IDeviceSoftwareWithUiStatus[],
  category_id?: number
): IDeviceSoftwareWithUiStatus[] => {
  // Find the category value string for the given id
  const category = CATEGORIES_NAV_ITEMS.find((cat) => cat.id === category_id);

  // If "All" is selected or category not found, return all software items
  if (!category || category.value === "All") {
    return software || [];
  }

  // Otherwise, filter software items whose categories include the category value
  return (software || []).filter(
    (softwareItem) =>
      softwareItem.software_package?.categories?.includes(
        category.value as SoftwareCategory
      ) ||
      softwareItem.app_store_app?.categories?.includes(
        category.value as SoftwareCategory
      )
  );
};

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
  if (!category) {
    return software;
  }
  const normalized = stripEmojiPrefix(category.name);
  return software.filter((item) => {
    const itemCategories = [
      ...(item.software_package?.categories ?? []),
      ...(item.app_store_app?.categories ?? []),
    ];
    return itemCategories.some((c) => c === normalized);
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

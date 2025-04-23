import { IDeviceSoftware, IHostSoftware } from "interfaces/software";
import { IGetDeviceSoftwareResponse } from "services/entities/device_user";

export interface ICategory {
  /** Temporary Clientside IDs */
  id: number;
  /** Text shown in the UI */
  label: string;
  /** Text stored in the API */
  value: string;
}

const ALL_ITEM = { id: 0, label: "All", value: "All" };

export const CATEGORIES_ITEMS = [
  { id: 1, label: "🌎 Browsers", value: "Browsers" },
  { id: 2, label: "👬 Communication", value: "Communication" },
  { id: 3, label: "🧰 Developer tools", value: "Developer tools" },
  { id: 4, label: "🖥️ Productivity", value: "Productivity" },
];

export const CATEGORIES_NAV_ITEMS = [ALL_ITEM, ...CATEGORIES_ITEMS];

export const filterSoftwareByCategory = (
  software?: IDeviceSoftware[],
  category_id?: number
): IDeviceSoftware[] => {
  // Find the category value string for the given id
  const category = CATEGORIES_NAV_ITEMS.find((cat) => cat.id === category_id);

  // If "All" is selected or category not found, return all software items
  if (!category || category.value === "All") {
    return software || [];
  }

  // Otherwise, filter software items whose categories include the category value
  return (software || []).filter((softwareItem) =>
    softwareItem.categories?.includes(category.value)
  );
};

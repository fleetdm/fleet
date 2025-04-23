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

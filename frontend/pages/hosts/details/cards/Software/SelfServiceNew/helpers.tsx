export interface ICategory {
  value?: number;
  label: string;
}

const CATEGORIES_NAV_ITEMS = [
  { value: undefined, label: "All" },
  { value: 1, label: "🌎 Browser" },
  { value: 2, label: "👬 Communication" },
  { value: 3, label: "🧰 Developer tools" },
  { value: 4, label: "🖥️ Productivity" },
];

export default CATEGORIES_NAV_ITEMS;

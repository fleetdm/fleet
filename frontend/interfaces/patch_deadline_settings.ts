export type Weekday =
  | "monday"
  | "tuesday"
  | "wednesday"
  | "thursday"
  | "friday"
  | "saturday"
  | "sunday";

export interface IPatchDeadlineSettings {
  fleet_id: number;
  days: Weekday[];
  // 24-hour clock, HH:MM.
  time: string;
}

// Canonical Monday-first weekday list. Single source of truth: the deadline
// form renders checkboxes from here, the deadline form sorts the PATCH payload
// using the same order, and the activity renderer uses WEEKDAY_ORDER to sort
// deadline-change activity copy for consistency.
export const WEEKDAYS: ReadonlyArray<{
  readonly key: Weekday;
  readonly label: string;
}> = [
  { key: "monday", label: "Monday" },
  { key: "tuesday", label: "Tuesday" },
  { key: "wednesday", label: "Wednesday" },
  { key: "thursday", label: "Thursday" },
  { key: "friday", label: "Friday" },
  { key: "saturday", label: "Saturday" },
  { key: "sunday", label: "Sunday" },
];

export const WEEKDAY_ORDER: Record<Weekday, number> = WEEKDAYS.reduce(
  (acc, w, i) => ({ ...acc, [w.key]: i }),
  {} as Record<Weekday, number>
);

const capitalizeWeekday = (w: Weekday): string =>
  w.charAt(0).toUpperCase() + w.slice(1);

// "Monday", "Monday and Friday", "Monday, Wednesday, and Friday". Drops any
// unknown weekday the server might send so the surrounding copy stays
// readable. Returns "" on an empty list; callers decide how to render that.
export const formatWeekdayList = (days: readonly string[]): string => {
  const known = days.filter((d): d is Weekday => d in WEEKDAY_ORDER);
  const sorted = [...known].sort((a, b) => WEEKDAY_ORDER[a] - WEEKDAY_ORDER[b]);
  const labels = sorted.map(capitalizeWeekday);
  if (labels.length === 0) return "";
  if (labels.length === 1) return labels[0];
  if (labels.length === 2) return `${labels[0]} and ${labels[1]}`;
  return `${labels.slice(0, -1).join(", ")}, and ${labels[labels.length - 1]}`;
};

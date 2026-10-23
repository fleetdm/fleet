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

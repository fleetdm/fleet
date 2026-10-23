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

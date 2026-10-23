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

// Software titles whose policy has patch_deadline=true for the queried fleet.
// Shape is a best-effort placeholder; swap to the real endpoint response once
// the backend lands. `name` is the canonical title (icon fallback matcher);
// `display_name` is the admin-overridable rename that must drive what the user
// sees — mirrors other software-title responses.
export interface IPatchDeadlineApp {
  id: number;
  name: string;
  display_name?: string;
  icon_url?: string | null;
  source?: string;
  bundle_identifier?: string;
}

export interface IPatchDeadlineAppsResponse {
  apps: IPatchDeadlineApp[];
  count: number;
}

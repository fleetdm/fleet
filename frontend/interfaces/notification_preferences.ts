// NotificationCategory mirrors the Go `fleet.NotificationCategory` enum. Keep
// the strings in sync with server/fleet/notification.go.
export type NotificationCategory =
  | "mdm"
  | "license"
  | "vulnerabilities"
  | "policies"
  | "software"
  | "hosts"
  | "integrations"
  | "system";

// NotificationChannel mirrors `fleet.NotificationChannel`. Only in_app is
// rendered today; email/slack exist so the prefs API can hold state ahead of
// the delivery workers landing.
export type NotificationChannel = "in_app" | "email" | "slack";

export interface INotificationPreference {
  category: NotificationCategory;
  channel: NotificationChannel;
  enabled: boolean;
}

export interface IListNotificationPreferencesResponse {
  preferences: INotificationPreference[];
}

export interface IUpdateNotificationPreferencesResponse {
  preferences: INotificationPreference[];
}

// Human-readable labels for the My Account UI. The Go side only emits stable
// identifiers; the UI owns the strings.
export const CATEGORY_LABELS: Record<
  NotificationCategory,
  { title: string; description: string }
> = {
  mdm: {
    title: "MDM",
    description:
      "APNs / ABM / VPP token expirations, Android Enterprise binding issues.",
  },
  license: {
    title: "License",
    description: "Fleet license expiration and seat-limit warnings.",
  },
  vulnerabilities: {
    title: "Vulnerabilities",
    description: "CVEs and CISA KEV matches affecting your fleet.",
  },
  policies: {
    title: "Policies",
    description: "Critical policies failing across many hosts.",
  },
  software: {
    title: "Software",
    description: "Install failure spikes and software inventory anomalies.",
  },
  hosts: {
    title: "Hosts",
    description: "Hosts offline, disk-encryption key escrow issues.",
  },
  integrations: {
    title: "Integrations",
    description: "Webhook delivery failures and integration health.",
  },
  system: {
    title: "System",
    description: "Fleet version updates and general system notices.",
  },
};

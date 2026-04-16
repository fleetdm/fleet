/**
 * Types for the in-app notification center (profile-avatar badge + modal).
 *
 * Distinct from `INotification` in `notification.ts`, which models transient
 * flash/toast messages rendered by `NotificationContext`. These types model
 * persistent, admin-facing, system-generated notifications served by
 * `GET /notifications`.
 */

export type NotificationSeverity = "error" | "warning" | "info";

/**
 * Stable IDs mirrored from `server/fleet/notification.go`. Treated as an open
 * string union so the frontend does not break if the backend adds a new type
 * the frontend does not yet know about.
 */
export type NotificationType =
  | "apns_cert_expiring"
  | "apns_cert_expired"
  | "abm_token_expiring"
  | "abm_token_expired"
  | "abm_terms_expired"
  | "vpp_token_expiring"
  | "vpp_token_expired"
  | "android_enterprise_deleted"
  | "license_expiring"
  | "license_expired"
  | string;

export interface INotificationCenterItem {
  id: number;
  type: NotificationType;
  severity: NotificationSeverity;
  title: string;
  body: string;
  cta_url?: string | null;
  cta_label?: string | null;
  metadata?: Record<string, unknown> | null;
  resolved_at?: string | null;
  created_at: string;
  updated_at: string;
  read_at?: string | null;
  dismissed_at?: string | null;
}

export interface IListNotificationsResponse {
  notifications: INotificationCenterItem[];
  unread_count: number;
}

export interface INotificationSummaryResponse {
  unread_count: number;
  active_count: number;
}

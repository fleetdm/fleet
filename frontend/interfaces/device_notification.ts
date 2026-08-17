// TODO(#50916): Remove this block once the BE endpoints in #50910 land and
// confirm the remaining assumptions.
//
// FE assumptions on the server contract (see #50916 issuecomment-5295478695):
//   1. Items carry their own `software_title_id` and `icon_url` — no second
//      software lookup on the client.
//   2. Both `org_logo_url_light_mode` and `org_logo_url_dark_mode` are always
//      set (BE falls back to Fleet's default logo when the admin hasn't
//      uploaded one). No deprecated aliases — this is a new endpoint.
//
// Confirmed by Rachel with the team on 2026-08-17:
//   * POST /device/:token/notifications/:uuid/actions returns the updated
//     NotificationView in the response body, so `update_now` transitions
//     to "Installing…" without a follow-up GET.
//   * 1-hour and 5-minute reminder share the same notification UUID; the
//     server flips its render output on an internal `reminder` flag
//     (#50912, #50913). The FE stays generic — no `kind` field needed.

export interface INotificationItem {
  software_title_id: number;
  /** Icon URL if present; <SoftwareIcon> falls back to `name` when null. */
  icon_url: string | null;
  /** Raw name — passed to <SoftwareIcon> unchanged for icon fallback matching. */
  name: string;
  display_name?: string;
  /** Optional right-aligned status label, e.g. "Installing…". */
  status?: string;
}

/** Server-declared actions rendered bottom-right. The last action is primary. */
export interface INotificationAction {
  /** Sent verbatim in the POST body, e.g. "update_now", "remind", "dismiss". */
  id: string;
  label: string;
}

export interface INotificationView {
  uuid: string;
  org_logo_url_light_mode: string;
  org_logo_url_dark_mode: string;
  /** May contain **bold** markup. */
  title: string;
  /** May contain **bold** markup. */
  description: string;
  items: INotificationItem[];
  actions: INotificationAction[];
}

/** JS → Swift bridge message ids. Distinct from server-side action ids. */
export type BridgeAction = "ready" | "resize" | "primary" | "dismiss" | "error";

/** Bump when the bridge envelope shape changes; Swift keys off `v` to route. */
export const BRIDGE_VERSION = 1 as const;

export interface IBridgeMessage {
  v: typeof BRIDGE_VERSION;
  action: BridgeAction;
  payload: Record<string, unknown>;
}

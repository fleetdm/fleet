// FE assumptions on the server contract, pending BE confirmation on #50916:
//   1. POST /device/:token/notifications/:uuid/actions returns the updated
//      INotificationView in the response body — so `update_now` returns the
//      "Installing…" state without a follow-up GET.
//   2. Items carry their own `software_title_id` and `icon_url` — no second
//      software lookup on the client.
//   3. Both `org_logo_url_light_mode` and `org_logo_url_dark_mode` are always
//      set (BE falls back to Fleet's default logo when the admin hasn't
//      uploaded one). No deprecated aliases — this is a new endpoint.

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

export interface IBridgeMessage {
  v: 1;
  action: BridgeAction;
  payload: Record<string, unknown>;
}

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

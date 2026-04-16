/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  IListNotificationsResponse,
  INotificationSummaryResponse,
} from "interfaces/notification_center";
import { buildQueryStringFromParams } from "utilities/url";

export interface IListNotificationsParams {
  include_dismissed?: boolean;
  include_resolved?: boolean;
}

export default {
  /**
   * List in-app notifications for the current user. Defaults to active,
   * non-dismissed, non-resolved notifications — the same set driving the
   * dropdown modal. Pass `include_dismissed` for the settings page view.
   */
  list: (
    params: IListNotificationsParams = {}
  ): Promise<IListNotificationsResponse> => {
    const { NOTIFICATIONS } = endpoints;
    const qs = buildQueryStringFromParams({
      include_dismissed: params.include_dismissed ? "true" : undefined,
      include_resolved: params.include_resolved ? "true" : undefined,
    });
    const path = qs ? `${NOTIFICATIONS}?${qs}` : NOTIFICATIONS;
    return sendRequest("GET", path);
  },

  /** Cheap count endpoint — drives the avatar badge. */
  summary: (): Promise<INotificationSummaryResponse> => {
    return sendRequest("GET", endpoints.NOTIFICATIONS_SUMMARY);
  },

  dismiss: (id: number): Promise<Record<string, never>> => {
    return sendRequest("PATCH", endpoints.NOTIFICATION_DISMISS(id));
  },

  restore: (id: number): Promise<Record<string, never>> => {
    return sendRequest("POST", endpoints.NOTIFICATION_RESTORE(id));
  },

  markRead: (id: number): Promise<Record<string, never>> => {
    return sendRequest("PATCH", endpoints.NOTIFICATION_READ(id));
  },

  markAllRead: (): Promise<Record<string, never>> => {
    return sendRequest("POST", endpoints.NOTIFICATIONS_READ_ALL);
  },
};

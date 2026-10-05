import { INotificationView } from "interfaces/device_notification";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IPostDeviceNotificationActionParams {
  deviceToken: string;
  notificationUuid: string;
  /** Action id declared by the server on INotificationAction, sent verbatim. */
  action: string;
}

export default {
  getNotification: (
    deviceToken: string,
    notificationUuid: string
  ): Promise<INotificationView> => {
    const { DEVICE_NOTIFICATION } = endpoints;
    return sendRequest(
      "GET",
      DEVICE_NOTIFICATION(deviceToken, notificationUuid)
    );
  },

  postNotificationAction: ({
    deviceToken,
    notificationUuid,
    action,
  }: IPostDeviceNotificationActionParams): Promise<INotificationView> => {
    const { DEVICE_NOTIFICATION_ACTIONS } = endpoints;
    return sendRequest(
      "POST",
      DEVICE_NOTIFICATION_ACTIONS(deviceToken, notificationUuid),
      { action }
    );
  },
};

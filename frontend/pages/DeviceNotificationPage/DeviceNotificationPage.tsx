import React, { useEffect, useRef } from "react";
import { useQuery } from "react-query";

import deviceNotificationsAPI from "services/entities/device_notifications";
import { INotificationView } from "interfaces/device_notification";

import { postBridgeMessage } from "./fleetDesktopBridge";

const baseClass = "device-notification-page";

interface IDeviceNotificationPageParams {
  device_auth_token: string;
  notification_uuid: string;
}

interface IDeviceNotificationPageProps {
  params: IDeviceNotificationPageParams;
}

const DeviceNotificationPage = ({
  params: { device_auth_token, notification_uuid },
}: IDeviceNotificationPageProps): JSX.Element | null => {
  const readyPostedRef = useRef(false);

  const { data, isSuccess, isError } = useQuery<INotificationView, Error>(
    ["device-notification", device_auth_token, notification_uuid],
    () =>
      deviceNotificationsAPI.getNotification(
        device_auth_token,
        notification_uuid
      ),
    {
      // One-shot fetch — the toast opens, fetches once, and is dismissed.
      // Server owns retries via the exit-code-driven notification pipeline.
      retry: false,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  useEffect(() => {
    if (isSuccess && !readyPostedRef.current) {
      readyPostedRef.current = true;
      // `ready` unblocks the ToastWindow.loadTimeout (30s in Fleet Desktop);
      // without it the notification script would report exit code 30.
      postBridgeMessage("ready");
    }
  }, [isSuccess]);

  useEffect(() => {
    if (isError) {
      postBridgeMessage("error");
    }
  }, [isError]);

  if (isError || !data) {
    return null;
  }

  return (
    <div className={baseClass}>
      <p>{data.title}</p>
      <p>{data.description}</p>
      <ul>
        {data.items.map((item) => (
          <li key={item.software_title_id}>
            {item.display_name ?? item.name}
            {item.status ? ` — ${item.status}` : ""}
          </li>
        ))}
      </ul>
      <div>
        {data.actions.map((action) => (
          <button key={action.id} type="button">
            {action.label}
          </button>
        ))}
      </div>
    </div>
  );
};

export default DeviceNotificationPage;

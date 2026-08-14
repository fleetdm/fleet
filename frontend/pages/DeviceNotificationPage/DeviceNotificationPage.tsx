import React from "react";

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
}: IDeviceNotificationPageProps): JSX.Element => {
  return (
    <div className={baseClass}>
      <p>Device: {device_auth_token}</p>
      <p>Notification: {notification_uuid}</p>
    </div>
  );
};

export default DeviceNotificationPage;

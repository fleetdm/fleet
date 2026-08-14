import React, { useEffect, useRef } from "react";
import { useQuery } from "react-query";

import Button from "components/buttons/Button";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

import deviceNotificationsAPI from "services/entities/device_notifications";
import {
  INotificationItem,
  INotificationView,
} from "interfaces/device_notification";

import { postBridgeMessage } from "./fleetDesktopBridge";

const baseClass = "device-notification-page";

interface IDeviceNotificationPageParams {
  device_auth_token: string;
  notification_uuid: string;
}

interface IDeviceNotificationPageProps {
  params: IDeviceNotificationPageParams;
}

// The server sends inline `**bold**` in title and description. This is the
// only markup, so a small splitter is cheaper than pulling in react-markdown.
const renderBoldMarkup = (text: string): React.ReactNode => {
  const parts = text.split(/(\*\*[^*]+\*\*)/g);
  return parts.map((part, i) => {
    if (part.startsWith("**") && part.endsWith("**")) {
      // eslint-disable-next-line react/no-array-index-key
      return <strong key={i}>{part.slice(2, -2)}</strong>;
    }
    // eslint-disable-next-line react/no-array-index-key
    return <React.Fragment key={i}>{part}</React.Fragment>;
  });
};

const NotificationItemRow = ({ item }: { item: INotificationItem }) => (
  <li className={`${baseClass}__item`}>
    <SoftwareIcon name={item.name} url={item.icon_url} />
    <span className={`${baseClass}__item-name`}>
      {item.display_name ?? item.name}
    </span>
    {item.status && (
      <span className={`${baseClass}__item-status`}>{item.status}</span>
    )}
  </li>
);

const DeviceNotificationPage = ({
  params: { device_auth_token, notification_uuid },
}: IDeviceNotificationPageProps): JSX.Element | null => {
  const readyPostedRef = useRef(false);
  const cardRef = useRef<HTMLDivElement>(null);

  const { data, isSuccess, isError } = useQuery<INotificationView, Error>(
    ["device-notification", device_auth_token, notification_uuid],
    () =>
      deviceNotificationsAPI.getNotification(
        device_auth_token,
        notification_uuid
      ),
    {
      retry: false,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  useEffect(() => {
    if (isSuccess && !readyPostedRef.current) {
      readyPostedRef.current = true;
      postBridgeMessage("ready");
    }
  }, [isSuccess]);

  useEffect(() => {
    if (isError) {
      postBridgeMessage("error");
    }
  }, [isError]);

  useEffect(() => {
    const node = cardRef.current;
    if (!node || !data) return undefined;
    // Native ToastWindow starts at 525x318 and resizes to fit the card. Every
    // height change (list scroll, action swap after update_now) posts back so
    // the window follows without an assumed fixed height.
    const observer = new ResizeObserver((entries) => {
      const height = entries[0]?.contentRect.height ?? 0;
      postBridgeMessage("resize", { height });
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, [data]);

  if (isError || !data) {
    return null;
  }

  const actions = data.actions;
  const primaryAction = actions[actions.length - 1];
  const secondaryActions = actions.slice(0, -1);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__card`} ref={cardRef}>
        <picture className={`${baseClass}__logo`}>
          <source
            srcSet={data.org_logo_url_dark_mode}
            media="(prefers-color-scheme: dark)"
          />
          <img src={data.org_logo_url_light_mode} alt="" />
        </picture>
        <p className={`${baseClass}__title`}>{renderBoldMarkup(data.title)}</p>
        <p className={`${baseClass}__description`}>
          {renderBoldMarkup(data.description)}
        </p>
        <ul className={`${baseClass}__item-list`}>
          {data.items.map((item) => (
            <NotificationItemRow key={item.software_title_id} item={item} />
          ))}
        </ul>
        <div className={`${baseClass}__actions`}>
          {secondaryActions.map((action) => (
            <Button key={action.id} variant="subdued">
              {action.label}
            </Button>
          ))}
          {primaryAction && (
            <Button key={primaryAction.id}>{primaryAction.label}</Button>
          )}
        </div>
      </div>
    </div>
  );
};

export default DeviceNotificationPage;

import React, { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import List from "components/List";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import deviceNotificationsAPI from "services/entities/device_notifications";
import {
  INotificationAction,
  INotificationItem,
  INotificationView,
} from "interfaces/device_notification";

import { isDarkMode } from "utilities/theme";

import { postBridgeMessage } from "./fleetDesktopBridge";

const baseClass = "device-notification-page";

interface IDeviceNotificationPageParams {
  device_auth_token: string;
  notification_uuid: string;
}

interface IDeviceNotificationPageProps {
  params: IDeviceNotificationPageParams;
}

// Title/description carry inline `**bold**` as the only markup. Avoid pulling
// in react-markdown for one construct.
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

const renderNotificationItemRow = (item: INotificationItem) => (
  <>
    <span className={`${baseClass}__item-left`}>
      <SoftwareIcon name={item.name} url={item.icon_url} size="small" />
      <span className={`${baseClass}__item-name`}>
        <TooltipTruncatedText
          value={getDisplayedSoftwareName(item.name, item.display_name)}
        />
      </span>
    </span>
    {item.status && (
      <span className={`${baseClass}__item-status`}>{item.status}</span>
    )}
  </>
);

const DeviceNotificationPage = ({
  params: { device_auth_token, notification_uuid },
}: IDeviceNotificationPageProps): JSX.Element | null => {
  const readyPostedRef = useRef(false);
  const cardRef = useRef<HTMLDivElement>(null);
  const queryClient = useQueryClient();
  const [darkMode, setDarkMode] = useState(() => isDarkMode());

  useEffect(() => {
    const onThemeChange = (e: Event) => {
      setDarkMode((e as CustomEvent).detail.dark);
    };
    window.addEventListener("fleet-theme-change", onThemeChange);
    return () =>
      window.removeEventListener("fleet-theme-change", onThemeChange);
  }, []);

  const queryKey = [
    "device-notification",
    device_auth_token,
    notification_uuid,
  ];

  const { data, isSuccess, isError } = useQuery<INotificationView, Error>(
    queryKey,
    () =>
      deviceNotificationsAPI.getNotification(
        device_auth_token,
        notification_uuid
      ),
    {
      // Keep refetchOnMount at its default: the toast reopens with a fresh
      // 30-min auth token, and a cached view from a previous open would be stale.
      retry: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  const {
    mutate: postAction,
    isError: isPostError,
    isLoading: isPosting,
  } = useMutation<
    INotificationView,
    Error,
    { action: INotificationAction; isPrimary: boolean }
  >(
    ({ action }) =>
      deviceNotificationsAPI.postNotificationAction({
        deviceToken: device_auth_token,
        notificationUuid: notification_uuid,
        action: action.id,
      }),
    {
      onSuccess: (updatedView, { action, isPrimary }) => {
        // Server returns the post-action view; write it back so we re-render
        // without a follow-up GET.
        queryClient.setQueryData(queryKey, updatedView);
        // `dismiss` id always closes, even when it's positionally primary
        // (e.g. the lone `Hide` in the Installing state).
        const shouldClose = action.id === "dismiss" || !isPrimary;
        postBridgeMessage(shouldClose ? "dismiss" : "primary");
      },
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
    // Use `offsetHeight` (border box) so the reported height includes the
    // card's padding; `contentRect.height` reports content box and clips buttons.
    const observer = new ResizeObserver(() => {
      postBridgeMessage("resize", { height: node.offsetHeight });
    });
    observer.observe(node);
    return () => observer.disconnect();
  }, [data]);

  if (isError || !data) {
    return null;
  }

  const view = data;
  const actions = view.actions;
  const primaryAction = actions[actions.length - 1];
  const secondaryActions = actions.slice(0, -1);
  const orgLogoURL = darkMode
    ? view.org_logo_url_dark_mode
    : view.org_logo_url_light_mode;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__card`} ref={cardRef}>
        <div className={`${baseClass}__header`}>
          <p className={`${baseClass}__title`}>
            {renderBoldMarkup(view.title)}
          </p>
          <p className={`${baseClass}__description`}>
            {renderBoldMarkup(view.description)}
          </p>
        </div>
        <List
          data={view.items}
          idKey="software_title_id"
          renderItemRow={renderNotificationItemRow}
        />
        <div className={`${baseClass}__actions`}>
          <OrgLogoIcon className={`${baseClass}__logo`} src={orgLogoURL} />
          {isPostError && (
            <div className={`${baseClass}__action-error`} role="alert">
              <DataError
                singleCustomLine
                description="Please try again."
                excludeIssueLink
              />
            </div>
          )}
          {secondaryActions.map((action) => (
            <Button
              key={action.id}
              variant="subdued"
              size="small"
              disabled={isPosting}
              onClick={() => postAction({ action, isPrimary: false })}
            >
              {action.label}
            </Button>
          ))}
          {primaryAction && (
            <Button
              key={primaryAction.id}
              size="small"
              disabled={isPosting}
              onClick={() =>
                postAction({ action: primaryAction, isPrimary: true })
              }
            >
              {primaryAction.label}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};

export default DeviceNotificationPage;

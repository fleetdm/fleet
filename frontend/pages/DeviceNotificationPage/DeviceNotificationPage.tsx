import React, { useEffect, useRef } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";

import Button from "components/buttons/Button";
import DataError from "components/DataError";
import List from "components/List";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

import deviceNotificationsAPI from "services/entities/device_notifications";
import {
  INotificationAction,
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

// Rendered as the child of Fleet's <List> row (`.list__row`), which owns the
// row's flex layout, padding, and dividers. We supply the left slot
// (icon + name) and the optional right slot (status), split by List's
// `justify-content: space-between`.
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
      // One-shot fetch — the toast opens, fetches once, and dismisses. Server
      // owns retries via exit codes. We DO want refetchOnMount at its default
      // (`true`) because the toast reopens every ~55 minutes with a fresh
      // device auth token (tokens rotate every 30m), and we should never
      // serve a cached view from a previous open.
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
        // Server owns the post-action state (e.g. Installing…). Replace the
        // cached view so re-render happens without a follow-up GET.
        queryClient.setQueryData(queryKey, updatedView);
        // `dismiss` id always closes the toast, regardless of position — the
        // server reuses it for any "close" action, including the lone `Hide`
        // in the Installing state where positionally it would otherwise be
        // the primary. Only real primary CTAs (e.g. `update_now`) keep the
        // window open.
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

  const view = data;
  const actions = view.actions;
  const primaryAction = actions[actions.length - 1];
  const secondaryActions = actions.slice(0, -1);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__card`} ref={cardRef}>
        <div className={`${baseClass}__header`}>
          <div className={`${baseClass}__header-text`}>
            <p className={`${baseClass}__title`}>
              {renderBoldMarkup(view.title)}
            </p>
            <p className={`${baseClass}__description`}>
              {renderBoldMarkup(view.description)}
            </p>
          </div>
          <picture className={`${baseClass}__logo`}>
            <source
              srcSet={view.org_logo_url_dark_mode}
              media="(prefers-color-scheme: dark)"
            />
            <img src={view.org_logo_url_light_mode} alt="" />
          </picture>
        </div>
        <List
          data={view.items}
          idKey="software_title_id"
          renderItemRow={renderNotificationItemRow}
        />
        <div className={`${baseClass}__actions`}>
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

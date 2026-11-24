import React, { useEffect, useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";

import {
  INotificationAction,
  INotificationView,
  isNotificationItemInstalling,
} from "interfaces/device_notification";
import deviceNotificationsAPI from "services/entities/device_notifications";
import { isDarkMode } from "utilities/theme";

import DeviceNotificationCard from "./DeviceNotificationCard";
import { postBridgeMessage } from "./fleetDesktopBridge";

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
      // Poll every 5 seconds while an install is pending.
      // Refetch the 5 minute reminder at install_at and then every 30 seconds, so the installs the cron queues within a minute of it show as pending before they finish.
      // Keep polling in the background because the toast window is rarely the focused one.
      refetchInterval: (view) => {
        if (view?.items.some(isNotificationItemInstalling)) {
          return 5000;
        }
        // Stop polling once the notification is acted on, Fleet removes Update now then.
        if (!view?.actions.some((action) => action.id === "update_now")) {
          return false;
        }
        // Skip polling on the 1 hour notice, Fleet shows the reminder in a new toast.
        if (view.actions.some((action) => action.id === "remind")) {
          return false;
        }
        if (!view.install_at) {
          return false;
        }
        const msUntilInstallAt =
          new Date(view.install_at).getTime() - Date.now();
        if (msUntilInstallAt <= 0) {
          return 30000;
        }
        return msUntilInstallAt;
      },
      refetchIntervalInBackground: true,
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
        // Swift's `case "primary"` fades the toast out, so ANY close-intent
        // action posts `dismiss`. `update_now` (primary, non-dismiss id) stays
        // open — ResizeObserver posts `resize` when the Installing view lands.
        const shouldClose = action.id === "dismiss" || !isPrimary;
        if (shouldClose) {
          postBridgeMessage("dismiss");
        }
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
  const orgLogoURL = darkMode
    ? view.org_logo_url_dark_mode
    : view.org_logo_url_light_mode;

  return (
    <DeviceNotificationCard
      view={view}
      orgLogoURL={orgLogoURL}
      onPostAction={(action, isPrimary) => postAction({ action, isPrimary })}
      isPosting={isPosting}
      isPostError={isPostError}
      cardRef={cardRef}
    />
  );
};

export default DeviceNotificationPage;

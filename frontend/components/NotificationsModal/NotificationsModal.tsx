import React, { useContext, useEffect, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "react-query";
import classnames from "classnames";

import notificationsAPI from "services/entities/notifications";
import {
  INotificationCenterItem,
  IListNotificationsResponse,
} from "interfaces/notification_center";
import { NotificationContext } from "context/notification";
import PATHS from "router/paths";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";

const baseClass = "notifications-modal";

/**
 * Query keys used by the notification center. Exported so the avatar badge
 * and other consumers invalidate the same cache entries.
 */
export const NOTIFICATIONS_QUERY_KEY = "notifications-list";
export const NOTIFICATIONS_SUMMARY_QUERY_KEY = "notifications-summary";

interface INotificationsModalProps {
  onExit: () => void;
  /**
   * Callback to navigate to an internal Fleet route (non-http URL from a CTA,
   * or the "View all notifications" button). The parent owns router access
   * so this component stays decoupled from `InjectedRouter`.
   */
  onNavigate: (path: string) => void;
}

const severityIcon = (severity: INotificationCenterItem["severity"]) => {
  switch (severity) {
    case "error":
      return "error-outline";
    case "warning":
      return "error-outline";
    default:
      return "info-outline";
  }
};

const NotificationsModal = ({
  onExit,
  onNavigate,
}: INotificationsModalProps): JSX.Element => {
  const queryClient = useQueryClient();
  const { renderFlash } = useContext(NotificationContext);

  const { data, isLoading, isError } = useQuery<
    IListNotificationsResponse,
    Error
  >([NOTIFICATIONS_QUERY_KEY, "active"], () => notificationsAPI.list(), {
    refetchOnWindowFocus: false,
  });

  const notifications = useMemo(() => data?.notifications ?? [], [data]);

  // Mark everything visible in the modal as read when it opens — matches how
  // most notification UIs behave. Individual dismiss is a separate action.
  useEffect(() => {
    if (notifications.length === 0) return;
    const anyUnread = notifications.some((n) => !n.read_at);
    if (!anyUnread) return;
    notificationsAPI.markAllRead().then(() => {
      queryClient.invalidateQueries([NOTIFICATIONS_SUMMARY_QUERY_KEY]);
    });
    // We intentionally depend on notifications.length — firing once per set of
    // notifications shown.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [notifications.length]);

  const dismissMutation = useMutation(
    (id: number) => notificationsAPI.dismiss(id),
    {
      onSuccess: () => {
        queryClient.invalidateQueries([NOTIFICATIONS_QUERY_KEY]);
        queryClient.invalidateQueries([NOTIFICATIONS_SUMMARY_QUERY_KEY]);
      },
      onError: () => {
        renderFlash("error", "Could not dismiss notification.");
      },
    }
  );

  const renderBody = () => {
    if (isLoading) {
      return <Spinner />;
    }
    if (isError) {
      return (
        <p className={`${baseClass}__empty`}>
          Could not load notifications. Try again later.
        </p>
      );
    }
    if (notifications.length === 0) {
      return (
        <p className={`${baseClass}__empty`}>
          You&apos;re all caught up. No active notifications.
        </p>
      );
    }
    return (
      <ul className={`${baseClass}__list`}>
        {notifications.map((n) => (
          <li
            key={n.id}
            className={classnames(
              `${baseClass}__item`,
              `${baseClass}__item--${n.severity}`
            )}
          >
            <div className={`${baseClass}__item-icon`}>
              <Icon name={severityIcon(n.severity)} size="small" />
            </div>
            <div className={`${baseClass}__item-content`}>
              <div className={`${baseClass}__item-title`}>{n.title}</div>
              <div className={`${baseClass}__item-body`}>{n.body}</div>
              <div className={`${baseClass}__item-actions`}>
                {n.cta_url && (
                  <Button
                    variant="text-icon"
                    onClick={() => {
                      if (!n.cta_url) return;
                      if (n.cta_url.startsWith("http")) {
                        window.open(n.cta_url, "_blank", "noopener,noreferrer");
                      } else {
                        onNavigate(n.cta_url);
                        onExit();
                      }
                    }}
                  >
                    {n.cta_label ?? "View"}
                  </Button>
                )}
                <Button
                  variant="text-icon"
                  onClick={() => dismissMutation.mutate(n.id)}
                  disabled={dismissMutation.isLoading}
                >
                  Dismiss
                </Button>
              </div>
            </div>
          </li>
        ))}
      </ul>
    );
  };

  const viewAll = () => {
    onNavigate(PATHS.NOTIFICATIONS);
    onExit();
  };

  return (
    <Modal
      title="Notifications"
      width="large"
      onExit={onExit}
      className={baseClass}
    >
      <>
        {renderBody()}
        <div className={`${baseClass}__footer`}>
          <Button variant="text-icon" onClick={viewAll}>
            View all notifications
          </Button>
          <Button variant="default" onClick={onExit}>
            Done
          </Button>
        </div>
      </>
    </Modal>
  );
};

export default NotificationsModal;

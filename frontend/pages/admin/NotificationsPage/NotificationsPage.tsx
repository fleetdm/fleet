import React, { useContext, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "react-query";
import { InjectedRouter } from "react-router";
import classnames from "classnames";

import notificationsAPI from "services/entities/notifications";
import {
  INotificationCenterItem,
  IListNotificationsResponse,
} from "interfaces/notification_center";
import { NotificationContext } from "context/notification";
import {
  NOTIFICATIONS_QUERY_KEY,
  NOTIFICATIONS_SUMMARY_QUERY_KEY,
} from "components/NotificationsModal";

import Button from "components/buttons/Button";
import Icon from "components/Icon";
import Spinner from "components/Spinner";
import DataError from "components/DataError";

const baseClass = "notifications-page";

interface INotificationsPageProps {
  router: InjectedRouter;
}

const severityIcon = (severity: INotificationCenterItem["severity"]) => {
  switch (severity) {
    case "error":
    case "warning":
      return "error-outline";
    default:
      return "info-outline";
  }
};

const NotificationsPage = ({
  router,
}: INotificationsPageProps): JSX.Element => {
  const queryClient = useQueryClient();
  const { renderFlash } = useContext(NotificationContext);
  const [showDismissed, setShowDismissed] = useState(false);

  const queryKey = [NOTIFICATIONS_QUERY_KEY, showDismissed ? "all" : "active"];

  const { data, isLoading, isError } = useQuery<
    IListNotificationsResponse,
    Error
  >(
    queryKey,
    () =>
      notificationsAPI.list({
        include_dismissed: showDismissed,
        include_resolved: false,
      }),
    { refetchOnWindowFocus: false }
  );

  const invalidate = () => {
    queryClient.invalidateQueries([NOTIFICATIONS_QUERY_KEY]);
    queryClient.invalidateQueries([NOTIFICATIONS_SUMMARY_QUERY_KEY]);
  };

  const dismissMutation = useMutation(
    (id: number) => notificationsAPI.dismiss(id),
    {
      onSuccess: invalidate,
      onError: () => renderFlash("error", "Could not dismiss notification."),
    }
  );

  const restoreMutation = useMutation(
    (id: number) => notificationsAPI.restore(id),
    {
      onSuccess: invalidate,
      onError: () => renderFlash("error", "Could not restore notification."),
    }
  );

  const renderEmpty = () => (
    <div className={`${baseClass}__empty`}>
      <p>
        {showDismissed
          ? "No notifications."
          : "You're all caught up. No active notifications."}
      </p>
    </div>
  );

  const renderList = (items: INotificationCenterItem[]) => (
    <ul className={`${baseClass}__list`}>
      {items.map((n) => {
        const dismissed = Boolean(n.dismissed_at);
        return (
          <li
            key={n.id}
            className={classnames(`${baseClass}__item`, {
              [`${baseClass}__item--${n.severity}`]: !dismissed,
              [`${baseClass}__item--dismissed`]: dismissed,
            })}
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
                        router.push(n.cta_url);
                      }
                    }}
                  >
                    {n.cta_label ?? "View"}
                  </Button>
                )}
                {dismissed ? (
                  <Button
                    variant="text-icon"
                    onClick={() => restoreMutation.mutate(n.id)}
                    disabled={restoreMutation.isLoading}
                  >
                    Restore
                  </Button>
                ) : (
                  <Button
                    variant="text-icon"
                    onClick={() => dismissMutation.mutate(n.id)}
                    disabled={dismissMutation.isLoading}
                  >
                    Dismiss
                  </Button>
                )}
              </div>
            </div>
          </li>
        );
      })}
    </ul>
  );

  const renderBody = () => {
    if (isLoading) return <Spinner />;
    if (isError) return <DataError />;
    const items = data?.notifications ?? [];
    if (items.length === 0) return renderEmpty();
    return renderList(items);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>
        <p className={`${baseClass}__description`}>
          System-generated notifications. Admins see these in the bell badge and
          dropdown.
        </p>
        <Button variant="text-icon" onClick={() => setShowDismissed((v) => !v)}>
          {showDismissed ? "Hide dismissed" : "Show dismissed"}
        </Button>
      </div>
      {renderBody()}
    </div>
  );
};

export default NotificationsPage;

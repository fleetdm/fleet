import React, { useEffect, useState, useRef } from "react";
import classnames from "classnames";

import { INotification } from "interfaces/notification";
import Icon from "components/Icon/Icon";

const baseClass = "flash-message";

export interface IFlashMessage {
  fullWidth: boolean;
  notification: INotification | INotification[] | null; // Handles single or multiple notifications
  isPersistent?: boolean;
  className?: string;
  onRemoveFlash: (id?: string) => void; // Accepts an id for removing specific notifications
  pathname?: string;
}

type ISingleFlashMessage = Omit<IFlashMessage, "notification"> & {
  notification: INotification;
};

// Component to render a single flash message
const SingleFlashMessage = ({
  notification,
  fullWidth,
  isPersistent,
  className,
  onRemoveFlash,
  pathname,
}: ISingleFlashMessage) => {
  const {
    alertType,
    isVisible,
    message,
    persistOnPageChange,
    id,
  } = notification;
  const baseClasses = classnames(
    baseClass,
    className,
    `${baseClass}--${alertType}`,
    {
      [`${baseClass}--full-width`]: fullWidth,
    }
  );

  const [hide, setHide] = useState(false);

  // This useEffect handles hiding successful flash messages after a 4s timeout.
  // By putting the notification in the dependency array, we can properly reset whenever a new flash message comes through.
  useEffect(() => {
    // Any time this hook runs, we reset the hide to false (so that subsequent messages that will be using this same component instance will be visible).
    setHide(false);

    if (!isPersistent && alertType === "success" && isVisible) {
      // After 4 seconds, set hide to true.
      const timer = setTimeout(() => {
        setHide(true);
        onRemoveFlash(); // This function resets notifications which allows CoreLayout reset of selected rows
      }, 4000);
      // Return a cleanup function that will clear this reset, in case another render happens after this.
      return () => clearTimeout(timer);
    }

    return undefined; // No cleanup when we don't set a timeout.
  }, [
    id,
    notification,
    alertType,
    isVisible,
    setHide,
    isPersistent,
    onRemoveFlash,
  ]);

  const isFirstRender = useRef(true);

  useEffect(() => {
    if (isFirstRender.current) {
      isFirstRender.current = false;
      return;
    }

    if (!persistOnPageChange) {
      setHide(true);
    }
  }, [pathname, persistOnPageChange]);

  if (hide || !isVisible) {
    return null;
  }

  return (
    <div className={baseClasses} id={baseClasses}>
      <div className={`${baseClass}__content`}>
        <Icon
          name={alertType === "success" ? "success" : "error"}
          color="core-fleet-white"
        />
        <span>{message}</span>
      </div>
      <div className={`${baseClass}__action`}>
        <div className={`${baseClass}__ex`}>
          <button
            className={`${baseClass}__remove ${baseClass}__remove--${alertType} button--unstyled`}
            onClick={() => onRemoveFlash(id)} // Pass the id to remove the specific flash message
          >
            <Icon
              name="close"
              color={
                alertType === "warning-filled"
                  ? "core-fleet-black"
                  : "core-fleet-white"
              }
            />
          </button>
        </div>
      </div>
    </div>
  );
};

const FlashMessage = ({
  fullWidth,
  notification,
  isPersistent,
  className,
  onRemoveFlash,
  pathname,
}: IFlashMessage): JSX.Element | null => {
  if (!notification) {
    return null; // Return null if there are no notifications
  }

  // Check if notification is an array and render accordingly
  if (Array.isArray(notification)) {
    const displayNotifications = notification.slice(0, 5); // Limit to 5 notifications
    return (
      <div className="flash-message-container">
        {displayNotifications.map((n) => (
          <SingleFlashMessage
            key={n.id}
            notification={n}
            fullWidth={fullWidth}
            isPersistent={isPersistent}
            className={className}
            onRemoveFlash={onRemoveFlash}
            pathname={pathname}
          />
        ))}
      </div>
    );
  }

  // Render a single notification if it's not an array
  return (
    <div className="flash-message-container">
      <SingleFlashMessage
        notification={notification}
        fullWidth={fullWidth}
        isPersistent={isPersistent}
        className={className}
        onRemoveFlash={onRemoveFlash}
        pathname={pathname}
      />
    </div>
  );
};

export default FlashMessage;

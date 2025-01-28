import React, { useEffect, useState } from "react";
import classnames from "classnames";

import { INotification } from "interfaces/notification";
// @ts-ignore
import Icon from "components/Icon/Icon";

const baseClass = "flash-message";

export interface IFlashMessage {
  fullWidth: boolean;
  notification: INotification | null;
  isPersistent?: boolean;
  className?: string;
  onRemoveFlash: () => void;
  pathname?: string;
}

const FlashMessage = ({
  fullWidth,
  notification,
  isPersistent,
  className,
  onRemoveFlash,
  pathname,
}: IFlashMessage): JSX.Element | null => {
  const { alertType, isVisible, message, persistOnPageChange } =
    notification || {};
  const baseClasses = classnames(
    baseClass,
    className,
    `${baseClass}--${alertType}`,
    {
      [`${baseClass}--full-width`]: fullWidth,
    }
  );

  const [hide, setHide] = useState(false);

  // This useEffect handles hiding successful flash messages after a 4s timeout. By putting the
  // notification in the dependency array, we can properly reset whenever a new flash message comes through.
  useEffect(() => {
    // Any time this hook runs, we reset the hide to false (so that subsequent messages that will be
    // using this same component instance will be visible).
    setHide(false);

    if (!isPersistent && alertType === "success" && isVisible) {
      // After 4 seconds, set hide to true.
      const timer = setTimeout(() => {
        setHide(true);
        onRemoveFlash(); // This function resets notifications which allows CoreLayout reset of selected rows
      }, 4000);
      // Return a cleanup function that will clear this reset, in case another render happens
      // after this. We want that render to set a new timeout (if needed).
      return () => clearTimeout(timer);
    }

    return undefined; // No cleanup when we don't set a timeout.
  }, [notification, alertType, isVisible, setHide]);

  useEffect(() => {
    if (!persistOnPageChange) {
      setHide(true);
    }
    // intentionally omit persistOnPageChange from dependencies to prevent hiding during initial
    // update of the notification prop from its default empty value
  }, [pathname]);

  if (hide || !isVisible) {
    return null;
  }

  return (
    <div className={"flash-message-container"}>
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
              onClick={onRemoveFlash}
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
    </div>
  );
};

export default FlashMessage;

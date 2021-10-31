import React, { useEffect, useState } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import { INotifications } from "interfaces/notification";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";

import CloseIcon from "../../../assets/images/icon-close-white-16x16@2x.png";
import CloseIconBlack from "../../../assets/images/icon-close-fleet-black-16x16@2x.png";

const baseClass = "flash-message";

export interface IFlashMessage {
  fullWidth: boolean;
  notification: INotifications;
  isPersistent?: boolean;
  onRemoveFlash: () => void;
  onUndoActionClick: (
    value: () => void
  ) => (evt: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
}

const FlashMessage = ({
  fullWidth,
  notification,
  isPersistent,
  onRemoveFlash,
  onUndoActionClick,
}: IFlashMessage) => {
  const { alertType, isVisible, message, undoAction } = notification;
  const klass = classnames(baseClass, `${baseClass}--${alertType}`, {
    [`${baseClass}--full-width`]: fullWidth,
  });

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

  if (hide || !isVisible) {
    return null;
  }

  const alertIcon =
    alertType === "success" ? "success-check" : "warning-filled";

  return (
    <div className={klass} id={klass}>
      <div className={`${baseClass}__content`}>
        <FleetIcon name={alertIcon} /> <span>{message}</span>
        {undoAction && (
          <Button
            className={`${baseClass}__undo`}
            variant="unstyled"
            onClick={onUndoActionClick(undoAction)}
          >
            Undo
          </Button>
        )}
      </div>
      <div className={`${baseClass}__action`}>
        <div className={`${baseClass}__ex`}>
          <button
            className={`${baseClass}__remove ${baseClass}__remove--${alertType} button--unstyled`}
            onClick={onRemoveFlash}
          >
            <img
              src={alertType === "warning-filled" ? CloseIconBlack : CloseIcon}
              alt="close icon"
            />
          </button>
        </div>
      </div>
    </div>
  );
};

export default FlashMessage;

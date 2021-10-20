import React, { useEffect, useState } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import { INotifications } from "interfaces/notification";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";

import CloseIcon from "../../../../assets/images/icon-close-white-16x16@2x.png";
import CloseIconBlack from "../../../../assets/images/icon-close-fleet-black-16x16@2x.png";

const baseClass = "flash-message";

interface IFlashMessage {
  fullWidth: boolean;
  notification: INotifications;
  onRemoveFlash: () => void;
  onUndoActionClick: (
    value: () => void
  ) => (evt: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
}

const FlashMessage = ({
  fullWidth,
  notification,
  onRemoveFlash,
  onUndoActionClick,
}: IFlashMessage) => {
  const { alertType, isVisible, message, undoAction } = notification;
  const klass = classnames(baseClass, `${baseClass}--${alertType}`, {
    [`${baseClass}--full-width`]: fullWidth,
  });

  if (alertType === "success") {
    setTimeout(() => {
      onRemoveFlash();
    }, 4000);
  }

  if (!isVisible) {
    return false;
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

import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import notificationInterface from "interfaces/notification";
import FleetIcon from "components/icons/FleetIcon";
import Button from "components/buttons/Button";

import CloseIcon from "../../../../assets/images/icon-close-white-16x16@2x.png";

const baseClass = "flash-message";

const FlashMessage = ({
  fullWidth,
  notification,
  onRemoveFlash,
  onUndoActionClick,
}) => {
  const { alertType, isVisible, message, undoAction } = notification;
  const klass = classnames(baseClass, `${baseClass}--${alertType}`, {
    [`${baseClass}--full-width`]: fullWidth,
  });

  if (!isVisible) {
    return false;
  }

  const alertIcon =
    alertType === "success" ? "success-check" : "warning-filled";

  return (
    <div className={klass}>
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
            <img src={CloseIcon} alt="close icon" />
          </button>
        </div>
      </div>
    </div>
  );
};

FlashMessage.propTypes = {
  fullWidth: PropTypes.bool,
  notification: notificationInterface,
  onRemoveFlash: PropTypes.func,
  onUndoActionClick: PropTypes.func,
};

export default FlashMessage;

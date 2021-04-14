import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import notificationInterface from "interfaces/notification";
import KolideIcon from "components/icons/KolideIcon";
import Button from "components/buttons/Button";

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
        <KolideIcon name={alertIcon} /> <span>{message}</span>
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
        <Button
          className={`${baseClass}__remove ${baseClass}__remove--${alertType}`}
          variant="unstyled"
          onClick={onRemoveFlash}
        >
          <KolideIcon name="x" />
        </Button>
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

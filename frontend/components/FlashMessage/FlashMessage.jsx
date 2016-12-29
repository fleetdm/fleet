import React, { PropTypes } from 'react';
import classnames from 'classnames';

import notificationInterface from '../../interfaces/notification';

const baseClass = 'flash-message';

const FlashMessage = ({ fullWidth, notification, onRemoveFlash, onUndoActionClick }) => {
  const { alertType, isVisible, message, undoAction } = notification;
  const klass = classnames(baseClass, `${baseClass}--${alertType}`, {
    [`${baseClass}--full-width`]: fullWidth,
  });

  if (!isVisible) {
    return false;
  }

  return (
    <div className={klass}>
      <div className={`${baseClass}__content`}>
        {message}
      </div>
      <div className={`${baseClass}__action`}>
        <button
          className={`${baseClass}__undo button button--unstyled`}
          onClick={onUndoActionClick(undoAction)}
        >
          {undoAction && 'undo'}
        </button>
        <button
          className={`${baseClass}__remove ${baseClass}__remove--${alertType} button button--unstyled`}
          onClick={onRemoveFlash}
        >
          &times;
        </button>
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

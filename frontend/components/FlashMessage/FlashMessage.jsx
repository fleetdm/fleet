import React, { PropTypes } from 'react';

import notificationInterface from '../../interfaces/notification';

const baseClass = 'flash-message';

const FlashMessage = ({ notification, onRemoveFlash, onUndoActionClick }) => {
  const { alertType, isVisible, message, undoAction } = notification;

  if (!isVisible) {
    return false;
  }

  return (
    <div className={`${baseClass} ${baseClass}--${alertType}`}>
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
  notification: notificationInterface,
  onRemoveFlash: PropTypes.func,
  onUndoActionClick: PropTypes.func,
};

export default FlashMessage;

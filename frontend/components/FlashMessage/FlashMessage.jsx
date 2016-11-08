import React, { PropTypes } from 'react';

import { hideFlash } from '../../redux/nodes/notifications/actions';
import notificationInterface from '../../interfaces/notification';

const baseClass = 'flash-message';

const FlashMessage = ({ notification, dispatch }) => {
  const { alertType, isVisible, message, undoAction } = notification;

  const submitUndoAction = () => {
    dispatch(undoAction);
    dispatch(hideFlash);
    return false;
  };

  const removeFlashMessage = () => {
    dispatch(hideFlash);
    return false;
  };

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
          onClick={submitUndoAction}
        >
          {undoAction && 'undo'}
        </button>
        <button
          className={`${baseClass}__remove ${baseClass}__remove--${alertType} button button--unstyled`}
          onClick={removeFlashMessage}
        >
          &times;
        </button>
      </div>
    </div>
  );
};

FlashMessage.propTypes = {
  dispatch: PropTypes.func,
  notification: notificationInterface,
};

export default FlashMessage;

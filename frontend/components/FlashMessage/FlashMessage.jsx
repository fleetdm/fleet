import React, { PropTypes } from 'react';
import radium from 'radium';
import componentStyles from './styles';
import { hideFlash } from '../../redux/nodes/notifications/actions';

const FlashMessage = ({ notification, dispatch }) => {
  const { alertType, isVisible, message, undoAction } = notification;
  const {
    containerStyles,
    contentStyles,
    flashActionStyles,
    removeFlashMessageStyles,
    undoStyles,
  } = componentStyles;

  const submitUndoAction = () => {
    dispatch(undoAction);
    dispatch(hideFlash);
    return false;
  };

  const removeFlashMessage = () => {
    dispatch(hideFlash);
    return false;
  };

  if (!isVisible) return false;

  return (
    <div style={containerStyles(alertType)}>
      <div style={contentStyles}>
        {message}
      </div>
      <div style={flashActionStyles}>
        <div onClick={submitUndoAction} style={undoStyles}>{undoAction && 'undo'}</div>
        <div onClick={removeFlashMessage} style={removeFlashMessageStyles(alertType)}>x</div>
      </div>
    </div>
  );
};

FlashMessage.propTypes = {
  dispatch: PropTypes.func,
  notification: PropTypes.object,
};

export default radium(FlashMessage);

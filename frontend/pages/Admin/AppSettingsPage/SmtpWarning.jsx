import React, { PropTypes } from 'react';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';

const baseClass = 'smtp-warning';

const SmtpWarning = ({ onDismiss, shouldShowWarning }) => {
  if (!shouldShowWarning) {
    return false;
  }

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__icon-wrap`}>
        <Icon name="warning-filled" />
        <span className={`${baseClass}__label`}>Warning!</span>
      </div>
      <span className={`${baseClass}__text`}>Email is not currently configured in Kolide. Many features rely on email to work.</span>
      <Button onClick={onDismiss} text="Dismiss" variant="unstyled" />
      <Button text="Resolve" variant="unstyled" />
    </div>
  );
};

SmtpWarning.propTypes = {
  onDismiss: PropTypes.func.isRequired,
  shouldShowWarning: PropTypes.bool.isRequired,
};

export default SmtpWarning;

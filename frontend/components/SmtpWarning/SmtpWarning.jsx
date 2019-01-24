import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';

const baseClass = 'smtp-warning';

const SmtpWarning = ({ className, onDismiss, onResolve, shouldShowWarning }) => {
  if (!shouldShowWarning) {
    return false;
  }

  const fullClassName = classnames(baseClass, className);

  return (
    <div className={fullClassName}>
      <div className={`${baseClass}__icon-wrap`}>
        <Icon name="warning-filled" />
        <span className={`${baseClass}__label`}>Warning!</span>
      </div>
      <span className={`${baseClass}__text`}>Email is not currently configured in Fleet. Many features rely on email to work.</span>
      {onDismiss && <Button onClick={onDismiss} variant="unstyled">Dismiss</Button>}
      {onResolve && <Button onClick={onResolve} variant="unstyled">Resolve</Button>}
    </div>
  );
};

SmtpWarning.propTypes = {
  className: PropTypes.string,
  onDismiss: PropTypes.func,
  onResolve: PropTypes.func,
  shouldShowWarning: PropTypes.bool.isRequired,
};

export default SmtpWarning;

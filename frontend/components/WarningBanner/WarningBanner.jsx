import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Button from 'components/buttons/Button';
import Icon from 'components/icons/Icon';

const baseClass = 'warning-banner';

const WarningBanner = ({ className, message, labelText, shouldShowWarning, onDismiss, onResolve }) => {
  if (!shouldShowWarning) {
    return false;
  }

  const fullClassName = classnames(baseClass, className);

  const label = labelText || 'Warning!';

  return (
    <div className={fullClassName}>
      <div className={`${baseClass}__icon-wrap`}>
        <Icon name="warning-filled" />
        <span className={`${baseClass}__label`}>{label}</span>
      </div>
      <span className={`${baseClass}__text`}>{message}</span>
      {onDismiss && <Button onClick={onDismiss} variant="unstyled">Dismiss</Button>}
      {onResolve && <Button onClick={onResolve} variant="unstyled">Resolve</Button>}
    </div>
  );
};

WarningBanner.propTypes = {
  className: PropTypes.string,
  message: PropTypes.string.isRequired,
  labelText: PropTypes.string,
  onDismiss: PropTypes.func,
  onResolve: PropTypes.func,
  shouldShowWarning: PropTypes.bool.isRequired,
};

export default WarningBanner;

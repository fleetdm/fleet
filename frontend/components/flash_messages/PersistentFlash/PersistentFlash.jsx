import React from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';

const baseClass = 'persistent-flash';

const PersistentFlash = ({ message }) => {
  const klass = classnames(baseClass, `${baseClass}--error`);

  return (
    <div className={klass}>
      <div className={`${baseClass}__content`}>
        <Icon name="warning-filled" /> <span>{message}</span>
      </div>
    </div>
  );
};

PersistentFlash.propTypes = {
  message: PropTypes.string.isRequired,
};

export default PersistentFlash;


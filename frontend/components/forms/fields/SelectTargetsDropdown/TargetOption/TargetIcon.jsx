import React from 'react';

import Icon from 'components/icons/Icon';
import targetInterface from 'interfaces/target';

const baseClass = 'target-option';

const TargetIcon = ({ target }) => {
  const iconName = () => {
    const { name, platform, target_type: targetType } = target;

    if (targetType === 'labels') {
      return name === 'All Hosts' ? 'all-hosts' : 'label';
    }

    return platform === 'darwin' ? 'apple' : platform;
  };

  return <Icon name={iconName()} className={`${baseClass}__icon`} />;
};

TargetIcon.propTypes = { target: targetInterface.isRequired };

export default TargetIcon;

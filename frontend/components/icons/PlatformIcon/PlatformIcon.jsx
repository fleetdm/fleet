import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

import Icon from 'components/icons/Icon';

const baseClass = 'platform-icon';

export class PlatformIcon extends Component {
  static propTypes = {
    className: PropTypes.string,
    fw: PropTypes.bool,
    name: PropTypes.string.isRequired,
    size: PropTypes.string,
    title: PropTypes.string,
  };

  findIcon = () => {
    const { name } = this.props;

    switch (name.toLowerCase()) {
      case 'macos': return 'apple';
      case 'mac os x': return 'apple';
      case 'mac osx': return 'apple';
      case 'mac os': return 'apple';
      case 'darwin': return 'apple';
      case 'centos': return 'centos';
      case 'centos linux': return 'centos';
      case 'ubuntu': return 'ubuntu';
      case 'ubuntu linux': return 'ubuntu';
      case 'linux': return 'linux';
      case 'windows': return 'windows';
      case 'ms windows': return 'windows';
      default: return false;
    }
  };

  render () {
    const { findIcon } = this;
    const { className, fw, name, size, title } = this.props;
    const iconClasses = classnames(baseClass, className);
    const iconName = findIcon();

    if (!iconName) {
      return <span className={iconClasses}>{name}</span>;
    }

    return (
      <Icon
        className={iconClasses}
        fw={fw}
        name={iconName}
        size={size}
        title={title}
      />
    );
  }
}

export default PlatformIcon;

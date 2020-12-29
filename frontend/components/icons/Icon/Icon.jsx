import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';

const baseClass = 'icon';

export class Icon extends Component {
  static propTypes = {
    className: PropTypes.string,
    name: PropTypes.string.isRequired,
    size: PropTypes.string.isRequired,
  };

  render () {
    const { className, name, size } = this.props;
    const src = `../../../assets/images/icon-${name}-${size}x${size}@2x.png`;
    const iconClasses = classnames(baseClass, className, {
      [`${baseClass}-${size}`]: size,
    });

    return (
      <img src={src} alt={`${name} icon`} className={iconClasses} />
    );
  }
}

export default Icon;

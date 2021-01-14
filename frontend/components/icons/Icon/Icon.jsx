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
    const requestImageFile = require.context('../../../../assets/images', true, /.png$/);
    const fileName = `icon-${name}-${size}x${size}@2x`;
    const iconClasses = classnames(baseClass, className, {
      [`${baseClass}-${size}`]: size,
    });

    return (
      <img src={requestImageFile(`./${fileName}.png`)} alt={`${name} icon`} className={iconClasses} />
    );
  }
}

export default Icon;

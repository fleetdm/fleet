import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';

const baseClass = 'shadow-box-input';

class ShadowBoxInput extends Component {
  static propTypes = {
    className: PropTypes.string,
    iconClass: PropTypes.string,
    name: PropTypes.string,
    placeholder: PropTypes.string,
  };

  render () {
    const { className, iconClass, name, placeholder } = this.props;
    const fullIconClassName = classnames(iconClass, `${baseClass}__icon`);
    const fullWrapperClassName = classnames(className, `${baseClass}__wrapper`);

    return (
      <div className={fullWrapperClassName}>
        <input
          className={`${baseClass}__input`}
          name={name}
          placeholder={placeholder}
        />
        {iconClass && <i className={fullIconClassName} />}
      </div>
    );
  }
}

export default ShadowBoxInput;

import React, { PropTypes } from 'react';
import classnames from 'classnames';

import InputField from '../InputField';

const baseClass = 'input-icon-field';

class InputFieldWithIcon extends InputField {
  static propTypes = {
    autofocus: PropTypes.bool,
    error: PropTypes.string,
    iconName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    placeholder: PropTypes.string,
    type: PropTypes.string,
  };

  renderHeading = () => {
    const { error, placeholder, value } = this.props;

    const labelClasses = classnames(
      `${baseClass}__label`,
      { [`${baseClass}__label--hidden`]: !value }
    );

    if (error) {
      return <div className={`${baseClass}__errors`}>{error}</div>;
    }

    return <div className={labelClasses}>{placeholder}</div>;
  }

  render () {
    const { error, iconName, name, placeholder, type, value } = this.props;
    const { onInputChange } = this;

    const inputClasses = classnames(
      `${baseClass}__input`,
      'input-with-icon',
      { [`${baseClass}__input--error`]: error },
      { [`${baseClass}__input--password`]: type === 'password' }
    );

    const iconClasses = classnames(
      `${baseClass}__icon`,
      iconName,
      { [`${baseClass}__icon--error`]: error },
      { [`${baseClass}__icon--active`]: value }
    );

    return (
      <div className={baseClass}>
        {this.renderHeading()}
        <input
          name={name}
          onChange={onInputChange}
          className={inputClasses}
          placeholder={placeholder}
          ref={(r) => { this.input = r; }}
          type={type}
          value={value}
        />
        {iconName && <i className={iconClasses} />}
      </div>
    );
  }
}

export default InputFieldWithIcon;

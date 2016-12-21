import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { noop, pick } from 'lodash';
import Select from 'react-select';

import dropdownOptionInterface from 'interfaces/dropdownOption';
import FormField from 'components/forms/FormField';

const baseClass = 'dropdown';

class Dropdown extends Component {
  static propTypes = {
    className: PropTypes.string,
    clearable: PropTypes.bool,
    error: PropTypes.string,
    label: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    labelClassName: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    options: PropTypes.arrayOf(dropdownOptionInterface).isRequired,
    placeholder: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    value: PropTypes.string,
    wrapperClassName: PropTypes.string,
  };

  static defaultProps = {
    onChange: noop,
    clearable: false,
    placeholder: 'Select One...',
  };

  handleChange = ({ value }) => {
    const { onChange } = this.props;

    return onChange(value);
  };

  renderLabel = () => {
    const { error, label, labelClassName, name } = this.props;
    const labelWrapperClasses = classnames(
      `${baseClass}__label`,
      labelClassName,
      { [`${baseClass}__label--error`]: error }
    );

    if (!label) {
      return false;
    }

    return (
      <label
        className={labelWrapperClasses}
        htmlFor={name}
      >
        {error || label}
      </label>
    );
  }

  render () {
    const { handleChange } = this;
    const { className, clearable, name, options, placeholder, value, wrapperClassName } = this.props;

    const formFieldProps = pick(this.props, ['hint', 'label', 'error', 'name']);

    return (
      <FormField {...formFieldProps} type="dropdown" className={wrapperClassName}>
        <Select
          className={`${baseClass}__select ${className}`}
          clearable={clearable}
          name={`${name}-select` || 'targets'}
          onChange={handleChange}
          options={options}
          placeholder={placeholder}
          value={value}
        />
      </FormField>
    );
  }
}

export default Dropdown;

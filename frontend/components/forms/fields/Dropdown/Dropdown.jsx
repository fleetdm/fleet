import React, { Component, PropTypes } from 'react';
import Select from 'react-select';
import { noop, pick } from 'lodash';

import dropdownOptionInterface from 'interfaces/dropdownOption';
import FormField from 'components/forms/FormField';

const baseClass = 'input-dropdown';

class Dropdown extends Component {
  static propTypes = {
    options: PropTypes.arrayOf(dropdownOptionInterface).isRequired,
    onChange: PropTypes.func,
    className: PropTypes.string,
    error: PropTypes.string,
    hint: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    label: PropTypes.string,
    placeholder: PropTypes.string,
    value: PropTypes.string,
    clearable: PropTypes.bool,
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

  render () {
    const { handleChange } = this;
    const { options, className, placeholder, value, clearable } = this.props;

    const formFieldProps = pick(this.props, ['hint', 'label', 'error', 'name']);

    return (
      <FormField {...formFieldProps} type="dropdown">
        <Select
          className={`${baseClass}__select ${className}`}
          name="targets"
          options={options}
          onChange={handleChange}
          placeholder={placeholder}
          value={value}
          clearable={clearable}
        />
      </FormField>
    );
  }
}

export default Dropdown;

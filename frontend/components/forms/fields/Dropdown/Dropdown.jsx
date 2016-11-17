import React, { Component, PropTypes } from 'react';
import Select from 'react-select';
import { noop } from 'lodash';

import dropdownOptionInterface from 'interfaces/dropdownOption';

class Dropdown extends Component {
  static propTypes = {
    options: PropTypes.arrayOf(dropdownOptionInterface).isRequired,
    onSelect: PropTypes.func,
    className: PropTypes.string,
    placeholder: PropTypes.string,
    value: PropTypes.string,
    clearable: PropTypes.bool,
  };

  static defaultProps = {
    onSelect: noop,
    clearable: false,
    placeholder: 'Select One...',
  };

  render () {
    const { options, className, placeholder, value, clearable, onSelect } = this.props;

    return (
      <Select
        className={className}
        name="targets"
        options={options}
        onChange={onSelect}
        placeholder={placeholder}
        value={value}
        clearable={clearable}
      />
    );
  }
}

export default Dropdown;

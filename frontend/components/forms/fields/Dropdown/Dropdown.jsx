import React, { Component, PropTypes } from 'react';
import { noop } from 'lodash';

import dropdownOptionInterface from '../../../../interfaces/dropdownOption';

const baseClass = 'kolide-dropdown';

class Dropdown extends Component {
  static propTypes = {
    options: PropTypes.arrayOf(dropdownOptionInterface),
    onSelect: PropTypes.func,
    className: PropTypes.string,
  };

  static defaultProps = {
    onSelect: noop,
  };

  onOptionClick = (evt) => {
    evt.preventDefault();

    const { onSelect } = this.props;

    onSelect(evt);

    return false;
  }

  renderOption = (option) => {
    const { disabled = false, value, text } = option;

    return (
      <option key={value} className={`${baseClass}__option`} value={value} disabled={disabled}>
        {text}
      </option>
    );
  }

  render () {
    const { options, className } = this.props;
    const { onOptionClick, renderOption } = this;

    return (
      <div className={[`${baseClass}__wrapper ${className}`]}>
        <select className={baseClass} onChange={onOptionClick}>
          {options.map((option) => {
            return renderOption(option);
          })}
        </select>
      </div>
    );
  }
}

export default Dropdown;

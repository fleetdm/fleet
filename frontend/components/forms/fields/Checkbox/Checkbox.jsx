import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { noop } from 'lodash';

const baseClass = 'kolide-checkbox';

class InputField extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
  };

  static defaultProps = {
    onChange: noop,
  };

  render () {
    const { children, className, name, onChange } = this.props;
    const checkBoxClass = classnames(baseClass, className);

    return (
      <label htmlFor={name} className={checkBoxClass}>
        <input type="checkbox" name={name} id={name} className={`${checkBoxClass}__input`} onChange={onChange} />
        <span className={`${checkBoxClass}__tick`} />
        {children}
      </label>
    );
  }
}

export default InputField;

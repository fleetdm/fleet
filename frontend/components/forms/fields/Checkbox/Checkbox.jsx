import React, { Component, PropTypes } from 'react';
import classnames from 'classnames';
import { noop, pick } from 'lodash';

import FormField from 'components/forms/FormField';

const baseClass = 'kolide-checkbox';

class Checkbox extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
    name: PropTypes.string,
    onChange: PropTypes.func,
    error: PropTypes.string,
    hint: PropTypes.oneOfType([PropTypes.array, PropTypes.string]),
    label: PropTypes.string,
    value: PropTypes.bool,
  };

  static defaultProps = {
    onChange: noop,
  };

  handleChange = () => {
    const { onChange, value } = this.props;

    return onChange(!value);
  };

  render () {
    const { handleChange } = this;
    const { children, className, name, value } = this.props;
    const checkBoxClass = classnames(baseClass, className);

    const formFieldProps = pick(this.props, ['hint', 'label', 'error', 'name']);

    return (
      <FormField {...formFieldProps} type="checkbox">
        <label htmlFor={name} className={checkBoxClass}>
          <input type="checkbox" name={name} id={name} className={`${checkBoxClass}__input`} onChange={handleChange} checked={value} />
          <span className={`${checkBoxClass}__tick`} />
          <div className={`${checkBoxClass}__label`}>{children}</div>
        </label>
      </FormField>
    );
  }
}

export default Checkbox;

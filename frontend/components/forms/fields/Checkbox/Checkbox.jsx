import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { noop, pick } from "lodash";

import FormField from "components/forms/FormField";

const baseClass = "kolide-checkbox";

class Checkbox extends Component {
  static propTypes = {
    children: PropTypes.node,
    className: PropTypes.string,
    disabled: PropTypes.bool,
    name: PropTypes.string,
    onChange: PropTypes.func,
    value: PropTypes.bool,
    wrapperClassName: PropTypes.string,
  };

  static defaultProps = {
    disabled: false,
    onChange: noop,
  };

  handleChange = () => {
    const { onChange, value } = this.props;

    return onChange(!value);
  };

  render() {
    const { handleChange } = this;
    const {
      children,
      className,
      disabled,
      name,
      value,
      wrapperClassName,
    } = this.props;
    const checkBoxClass = classnames(baseClass, className);

    const formFieldProps = pick(this.props, ["hint", "label", "error", "name"]);

    const checkBoxTickClass = classnames(`${checkBoxClass}__tick`, {
      [`${checkBoxClass}__tick--disabled`]: disabled,
    });

    return (
      <FormField
        {...formFieldProps}
        className={wrapperClassName}
        type="checkbox"
      >
        <label htmlFor={name} className={checkBoxClass}>
          <input
            checked={value}
            className={`${checkBoxClass}__input`}
            disabled={disabled}
            id={name}
            name={name}
            onChange={handleChange}
            type="checkbox"
          />
          <span className={checkBoxTickClass} />
          <span className={`${checkBoxClass}__label`}>{children}</span>
        </label>
      </FormField>
    );
  }
}

export default Checkbox;

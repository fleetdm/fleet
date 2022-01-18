import React from "react";
import classnames from "classnames";
import { noop, pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";

const baseClass = "fleet-checkbox";

export interface ICheckboxProps {
  children?: JSX.Element | Array<JSX.Element> | string;
  className?: string;
  disabled?: boolean;
  name?: string;
  onChange?: any; // TODO: meant to be an event; figure out type for this
  onBlur?: any;
  onFocus?: any;
  value?: boolean;
  wrapperClassName?: string;
  indeterminate?: boolean;
  parseTarget?: boolean;
}

const Checkbox = (props: ICheckboxProps) => {
  const {
    children,
    className,
    disabled = false,
    name,
    onChange = noop,
    onBlur = noop,
    onFocus = noop,
    value,
    wrapperClassName,
    indeterminate,
    parseTarget,
  } = props;

  const handleChange = () => {
    if (parseTarget) {
      // Returns both name and value
      return onChange({ name, value: !value });
    }

    return onChange(!value);
  };

  const checkBoxClass = classnames(baseClass, className);
  const formFieldProps = {
    ...pick(props, ["hint", "label", "error", "name"]),
    className: wrapperClassName,
    type: "checkbox",
  } as IFormFieldProps;

  const checkBoxTickClass = classnames(`${checkBoxClass}__tick`, {
    [`${checkBoxClass}__tick--disabled`]: disabled,
    [`${checkBoxClass}__tick--indeterminate`]: indeterminate,
  });

  return (
    <FormField {...formFieldProps}>
      <label htmlFor={name} className={checkBoxClass}>
        <input
          checked={value}
          className={`${checkBoxClass}__input`}
          disabled={disabled}
          id={name}
          name={name}
          onChange={handleChange}
          onBlur={onBlur}
          onFocus={onFocus}
          type="checkbox"
          ref={(element) => {
            element && indeterminate && (element.indeterminate = indeterminate);
          }}
        />
        <span className={checkBoxTickClass} />
        <span className={`${checkBoxClass}__label`}>{children}</span>
      </label>
    </FormField>
  );
};

export default Checkbox;

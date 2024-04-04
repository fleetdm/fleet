import React, { ReactNode } from "react";
import classnames from "classnames";
import { noop, pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "fleet-checkbox";

export interface ICheckboxProps {
  children?: ReactNode;
  className?: string;
  disabled?: boolean;
  name?: string;
  onChange?: any; // TODO: meant to be an event; figure out type for this
  onBlur?: any;
  value?: boolean;
  wrapperClassName?: string;
  indeterminate?: boolean;
  parseTarget?: boolean;
  tooltipContent?: React.ReactNode;
  isLeftLabel?: boolean;
  helpText?: React.ReactNode;
}

const Checkbox = (props: ICheckboxProps) => {
  const {
    children,
    className,
    disabled = false,
    name,
    onChange = noop,
    onBlur = noop,
    value,
    wrapperClassName,
    indeterminate,
    parseTarget,
    tooltipContent,
    isLeftLabel,
    helpText,
  } = props;

  const handleChange = () => {
    if (parseTarget) {
      // Returns both name and value
      return onChange({ name, value: !value });
    }

    return onChange(!value);
  };

  const checkBoxClass = classnames(
    { inverse: isLeftLabel },
    className,
    baseClass
  );

  const checkBoxTickClass = classnames(`${baseClass}__tick`, {
    [`${baseClass}__tick--disabled`]: disabled,
    [`${baseClass}__tick--indeterminate`]: indeterminate,
  });

  const checkBoxLabelClass = classnames(checkBoxClass, {
    [`${baseClass}__label--disabled`]: disabled,
  });

  const formFieldProps = {
    ...pick(props, ["helpText", "label", "error", "name"]),
    className: wrapperClassName,
    type: "checkbox",
  } as IFormFieldProps;

  return (
    <FormField {...formFieldProps}>
      <>
        <label htmlFor={name} className={checkBoxLabelClass}>
          <input
            checked={value}
            className={`${baseClass}__input`}
            disabled={disabled}
            id={name}
            name={name}
            onChange={handleChange}
            onBlur={onBlur}
            type="checkbox"
          />
          <span className={checkBoxTickClass} />
          {tooltipContent ? (
            <span className={`${baseClass}__label-tooltip tooltip`}>
              <TooltipWrapper tipContent={tooltipContent}>
                {children}
              </TooltipWrapper>
            </span>
          ) : (
            <span className={`${baseClass}__label`}>{children} </span>
          )}
        </label>
      </>
    </FormField>
  );
};

export default Checkbox;

import React, { ReactNode } from "react";
import classnames from "classnames";
import { noop, pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";
import TooltipWrapper from "components/TooltipWrapper";
import { ITooltipWrapperTipContent } from "components/TooltipWrapper/TooltipWrapper";

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
  tooltipContent?: ITooltipWrapperTipContent["tipContent"];
  isLeftLabel?: boolean;
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
              {children as string}
            </TooltipWrapper>
          </span>
        ) : (
          <span className={`${baseClass}__label`}>{children} </span>
        )}
      </label>
    </FormField>
  );
};

export default Checkbox;

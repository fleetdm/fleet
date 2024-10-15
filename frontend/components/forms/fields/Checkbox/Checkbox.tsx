import React, { ReactNode, KeyboardEvent } from "react";
import classnames from "classnames";
import { noop, pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";
import TooltipWrapper from "components/TooltipWrapper";
import Icon from "components/Icon";

const baseClass = "fleet-checkbox";

export interface ICheckboxProps {
  children?: ReactNode;
  className?: string;
  /** readOnly displays a non-editable field */
  readOnly?: boolean;
  /** disabled displays a greyed out non-editable field */
  disabled?: boolean;
  name?: string;
  onChange?: any; // TODO: meant to be an event; figure out type for this
  onBlur?: (event: React.FocusEvent<HTMLDivElement>) => void;
  value?: boolean | null;
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
    readOnly = false,
    disabled = false,
    name,
    onChange = noop,
    onBlur = noop,
    value = false,
    wrapperClassName,
    indeterminate = false,
    parseTarget,
    tooltipContent,
    isLeftLabel,
    helpText,
  } = props;

  const handleChange = (
    event: React.MouseEvent | React.KeyboardEvent
  ): void => {
    event.preventDefault();
    if (readOnly || disabled) return;

    let newValue: boolean | null;
    if (indeterminate) {
      newValue = true;
    } else if (value === true) {
      newValue = false;
    } else {
      newValue = true;
    }

    if (parseTarget) {
      onChange({ name, value: newValue });
    } else {
      onChange(newValue);
    }
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    if (event.key === "Enter" || event.key === " ") {
      handleChange(event);
    }
  };

  const checkBoxClass = classnames(
    { inverse: isLeftLabel },
    className,
    baseClass
  );

  const checkBoxLabelClass = classnames(checkBoxClass, {
    [`${baseClass}__label--read-only`]: readOnly || disabled,
    [`${baseClass}__label--disabled`]: disabled,
  });

  const formFieldProps = {
    ...pick(props, ["helpText", "label", "error", "name"]),
    className: wrapperClassName,
    type: "checkbox",
  } as IFormFieldProps;

  const getIconName = () => {
    if (indeterminate) return "checkbox-indeterminate";
    if (value) return "checkbox";
    return "checkbox-unchecked";
  };

  return (
    <FormField {...formFieldProps}>
      <div
        role="checkbox"
        aria-checked={indeterminate ? "mixed" : value || undefined}
        aria-readonly={readOnly}
        aria-disabled={disabled}
        tabIndex={disabled ? -1 : 0}
        className={checkBoxLabelClass}
        onClick={handleChange}
        onKeyDown={handleKeyDown}
        onBlur={onBlur}
      >
        <Icon
          name={getIconName()}
          className={`${baseClass}__icon ${baseClass}__icon--${getIconName()}`}
        />
        {tooltipContent ? (
          <span className={`${baseClass}__label-tooltip tooltip`}>
            <TooltipWrapper tipContent={tooltipContent} clickable={false}>
              {children}
            </TooltipWrapper>
          </span>
        ) : (
          <span className={`${baseClass}__label`}>{children}</span>
        )}
      </div>
    </FormField>
  );
};

export default Checkbox;

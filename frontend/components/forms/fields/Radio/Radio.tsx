import React, { ReactNode } from "react";
import classnames from "classnames";

import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "radio";

export interface IRadioProps {
  label: ReactNode;
  value: string;
  id: string;
  onChange: (value: string) => void;
  checked?: boolean;
  /** Allows tabbing to the group, then using arrow keys to move between options with same name attribute */
  name?: string;
  className?: string;
  disabled?: boolean;
  tooltip?: React.ReactNode;
  helpText?: React.ReactNode;
  testId?: string;
}

const Radio = ({
  className,
  id,
  name,
  value,
  checked,
  disabled,
  label,
  tooltip,
  helpText,
  testId,
  onChange,
}: IRadioProps): JSX.Element => {
  const wrapperClasses = classnames(baseClass, className, {
    [`${baseClass}__disabled`]: disabled,
  });

  return (
    <div className={wrapperClasses} data-testid={testId}>
      <label htmlFor={id} className={`${baseClass}__radio-control`}>
        <span className={`${baseClass}__input`}>
          <input
            type="radio"
            id={id}
            disabled={disabled}
            name={name}
            value={value}
            checked={checked}
            onChange={(event) => onChange(event.target.value)}
          />
          <span className={`${baseClass}__control-button`} />
        </span>
        <span className={`${baseClass}__label`}>
          {tooltip ? (
            <TooltipWrapper tipContent={tooltip}>{label}</TooltipWrapper>
          ) : (
            <>{label}</>
          )}
        </span>
      </label>
      {helpText && <div className={`${baseClass}__help-text`}>{helpText}</div>}
    </div>
  );
};

export default Radio;

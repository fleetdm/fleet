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
  name?: string;
  className?: string;
  disabled?: boolean;
  tooltip?: React.ReactNode;
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
  testId,
  onChange,
}: IRadioProps): JSX.Element => {
  const wrapperClasses = classnames(baseClass, className, {
    [`disabled`]: disabled,
  });

  return (
    <label htmlFor={id} className={wrapperClasses} data-testid={testId}>
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
        <span className={`${baseClass}__control`} />
      </span>
      <span className={`${baseClass}__label`}>
        {tooltip ? (
          <TooltipWrapper tipContent={tooltip}>{label}</TooltipWrapper>
        ) : (
          <>{label}</>
        )}
      </span>
    </label>
  );
};

export default Radio;

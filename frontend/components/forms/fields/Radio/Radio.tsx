import React from "react";
import classnames from "classnames";

const baseClass = "radio";

export interface IRadioProps {
  label: string;
  value: string;
  id: string;
  onChange: (value: string) => void;
  checked?: boolean;
  name?: string;
  className?: string;
  disabled?: boolean;
}

const Radio = ({
  className,
  id,
  name,
  value,
  checked,
  disabled,
  label,
  onChange,
}: IRadioProps): JSX.Element => {
  const wrapperClasses = classnames(baseClass, className);

  const radioControlClass = classnames({
    [`disabled`]: disabled,
  });

  return (
    <label htmlFor={id} className={`${wrapperClasses} ${radioControlClass}`}>
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
      <span className={`${baseClass}__label`}>{label}</span>
    </label>
  );
};

export default Radio;

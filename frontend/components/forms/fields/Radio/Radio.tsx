import React from 'react';
import classnames from 'classnames';

const baseClass = 'radio-input';

interface IRadioProps {
  label: string;
  value: string;
  id: string;
  onChange: (value: string) => void;
  checked?: boolean;
  name?: string;
  className?: string;
  disabled?: boolean;
}

const Radio = (props: IRadioProps): JSX.Element => {
  const { className, id, name, value, checked, disabled, label, onChange } = props;

  const wrapperClasses = classnames(baseClass, className);
  const checkBoxTickClass = classnames(`${wrapperClasses}__tick`, {
    [`${wrapperClasses}__tick--disabled`]: disabled,
  });

  return (
    <div className={wrapperClasses}>
      <label htmlFor={id}>
        <input
          type="radio"
          id={id}
          disabled={disabled}
          name={name}
          value={value}
          checked={checked}
          onChange={event => onChange(event.target.value)}
        />
        <span className={checkBoxTickClass} />
        <span className={`${wrapperClasses}__label`}>{label}</span>
      </label>
    </div>
  );
};

export default Radio;

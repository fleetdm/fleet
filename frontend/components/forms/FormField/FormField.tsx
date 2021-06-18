import React from "react";
import classnames from "classnames";
import { isEmpty } from "lodash";

const baseClass = "form-field";

export interface IFormFieldProps {
  children: JSX.Element;
  className: string;
  error: string;
  hint: Array<any> | JSX.Element | string;
  label: Array<any> | JSX.Element | string;
  name: string;
  type: string;
}

const FormField = ({
  children,
  className,
  error,
  hint,
  label,
  name,
  type,
}: IFormFieldProps) => {
  const renderLabel = () => {
    const labelWrapperClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !isEmpty(error),
    });

    if (!label) {
      return false;
    }

    return (
      <label className={labelWrapperClasses} htmlFor={name}>
        {error || label}
      </label>
    );
  };

  const renderHint = () => {
    if (hint) {
      return <span className={`${baseClass}__hint`}>{hint}</span>;
    }

    return false;
  };

  const formFieldClass = classnames(
    baseClass,
    {
      [`${baseClass}--${type}`]: !isEmpty(type),
    },
    className
  );

  return (
    <div className={formFieldClass}>
      {renderLabel()}
      {children}
      {renderHint()}
    </div>
  );
};

export default FormField;

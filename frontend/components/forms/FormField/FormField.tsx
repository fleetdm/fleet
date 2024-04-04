import React from "react";
import classnames from "classnames";
import { isEmpty } from "lodash";

import TooltipWrapper from "components/TooltipWrapper";
import { PlacesType } from "react-tooltip-5";

// all form-field styles are defined in _global.scss, which apply here and elsewhere
const baseClass = "form-field";

export interface IFormFieldProps {
  children: JSX.Element;
  className: string;
  error: string;
  helpText: Array<any> | JSX.Element | string;
  label: Array<any> | JSX.Element | string;
  name: string;
  type: string;
  tooltip?: React.ReactNode;
  labelTooltipPosition?: PlacesType;
}

const FormField = ({
  children,
  className,
  error,
  helpText,
  label,
  name,
  type,
  tooltip,
  labelTooltipPosition,
}: IFormFieldProps): JSX.Element => {
  const renderLabel = () => {
    const labelWrapperClasses = classnames(`${baseClass}__label`, {
      [`${baseClass}__label--error`]: !isEmpty(error),
    });

    if (!label) {
      return false;
    }

    return (
      <label
        className={labelWrapperClasses}
        htmlFor={name}
        data-has-tooltip={!!tooltip}
      >
        {error ||
          (tooltip ? (
            <TooltipWrapper
              tipContent={tooltip}
              position={labelTooltipPosition || "top-start"}
            >
              {label as string}
            </TooltipWrapper>
          ) : (
            <>{label}</>
          ))}
      </label>
    );
  };

  const renderHelpText = () => {
    if (helpText) {
      return <span className={`${baseClass}__help-text`}>{helpText}</span>;
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
      {renderHelpText()}
    </div>
  );
};

export default FormField;

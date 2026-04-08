import React, { useEffect, useRef } from "react";
import classnames from "classnames";
import { pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";
import TooltipWrapper from "components/TooltipWrapper";

interface ISliderProps {
  onChange: () => void;
  value: boolean;
  inactiveText: JSX.Element | string;
  activeText: JSX.Element | string;
  /** Use to display slider label tooltip, compatible with disabled state */
  labelTooltip?: JSX.Element | string;
  className?: string;
  helpText?: JSX.Element | string;
  autoFocus?: boolean;
  disabled?: boolean;
}

const baseClass = "fleet-slider";

const Slider = (props: ISliderProps): JSX.Element => {
  const {
    onChange,
    value,
    inactiveText,
    activeText,
    labelTooltip,
    autoFocus,
    disabled,
  } = props;

  const sliderRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (autoFocus && sliderRef.current) {
      sliderRef.current.focus();
    }
  }, [autoFocus]);

  const sliderBtnClass = classnames(baseClass, {
    [`${baseClass}--active`]: value,
  });

  const sliderDotClass = classnames(`${baseClass}__dot`, {
    [`${baseClass}__dot--active`]: value,
  });

  const handleClick = (evt: React.MouseEvent) => {
    evt.preventDefault();
    if (disabled) return;

    onChange();
  };

  const formFieldProps = pick(props, [
    "helpText",
    "label",
    "error",
    "name",
    "className",
    "disabled",
  ]) as IFormFieldProps;

  const wrapperClassNames = classnames(`${baseClass}__wrapper`, {
    [`${baseClass}__wrapper--disabled`]: disabled,
  });

  const text = value ? activeText : inactiveText;
  return (
    <FormField {...formFieldProps} type="slider">
      <div className={wrapperClassNames}>
        <button
          role="switch"
          aria-checked={value}
          className={`button button--unstyled ${sliderBtnClass}`}
          onClick={handleClick}
          disabled={disabled}
          ref={sliderRef}
        >
          <div className={sliderDotClass} />
        </button>
        {labelTooltip ? (
          <TooltipWrapper tipContent={labelTooltip}>
            {" "}
            <span
              className={`${baseClass}__label ${baseClass}__label--${
                value ? "active" : "inactive"
              }`}
            >
              {text}
            </span>
          </TooltipWrapper>
        ) : (
          <span
            className={`${baseClass}__label ${baseClass}__label--${
              value ? "active" : "inactive"
            }`}
          >
            {text}
          </span>
        )}
      </div>
    </FormField>
  );
};

export default Slider;

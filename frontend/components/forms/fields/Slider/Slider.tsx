import React from "react";
import classnames from "classnames";
import { pick } from "lodash";

import FormField from "components/forms/FormField";
import { IFormFieldProps } from "components/forms/FormField/FormField";

interface ISliderProps {
  onChange: () => void;
  value: boolean;
  inactiveText: JSX.Element | string;
  activeText: JSX.Element | string;
  className?: string;
  helpText?: JSX.Element | string;
}

const baseClass = "fleet-slider";

const Slider = (props: ISliderProps): JSX.Element => {
  const { onChange, value, inactiveText, activeText } = props;

  const sliderBtnClass = classnames(baseClass, {
    [`${baseClass}--active`]: value,
  });

  const sliderDotClass = classnames(`${baseClass}__dot`, {
    [`${baseClass}__dot--active`]: value,
  });

  const handleClick = (evt: React.MouseEvent) => {
    evt.preventDefault();

    return onChange();
  };

  const formFieldProps = pick(props, [
    "helpText",
    "label",
    "error",
    "name",
    "className",
  ]) as IFormFieldProps;

  return (
    <FormField {...formFieldProps} type="slider">
      <div className={`${baseClass}__wrapper`}>
        <button
          className={`button button--unstyled ${sliderBtnClass}`}
          onClick={handleClick}
        >
          <div className={sliderDotClass} />
        </button>
        <span
          className={`${baseClass}__label ${baseClass}__label--${
            value ? "active" : "inactive"
          }`}
        >
          {value ? activeText : inactiveText}
        </span>
      </div>
    </FormField>
  );
};

export default Slider;

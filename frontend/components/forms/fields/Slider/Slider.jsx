import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { pick } from "lodash";

import FormField from "components/forms/FormField";

const Slider = (props) => {
  const { onChange, value, inactiveText = "Off", activeText = "On" } = props;
  const baseClass = "fleet-slider";

  const sliderBtnClass = classnames(baseClass, {
    [`${baseClass}--active`]: value,
  });

  const sliderDotClass = classnames(`${baseClass}__dot`, {
    [`${baseClass}__dot--active`]: value,
  });

  const handleClick = (evt) => {
    evt.preventDefault();

    return onChange(!value);
  };

  const formFieldProps = pick(props, ["hint", "label", "error", "name"]);

  return (
    <FormField {...formFieldProps} type="slider">
      <div className={`${baseClass}__wrapper`}>
        <span className={`${baseClass}__label ${baseClass}__label--inactive`}>
          {inactiveText}
        </span>
        <button
          className={`button button--unstyled ${sliderBtnClass}`}
          onClick={handleClick}
        >
          <div className={sliderDotClass} />
        </button>
        <span className={`${baseClass}__label ${baseClass}__label--active`}>
          {activeText}
        </span>
      </div>
    </FormField>
  );
};

Slider.propTypes = {
  value: PropTypes.bool,
  onChange: PropTypes.func,
  inactiveText: PropTypes.string,
  activeText: PropTypes.string,
};

export default Slider;

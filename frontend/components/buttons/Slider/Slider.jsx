import React, { PropTypes } from 'react';
import classnames from 'classnames';

const Slider = ({ onChange, value }) => {
  const baseClass = 'slider-wrap';

  const sliderBtnClass = classnames(
    baseClass,
    { [`${baseClass}--active`]: value }
  );

  const sliderDotClass = classnames(
    `${baseClass}__dot`,
    { [`${baseClass}__dot--active`]: value }
  );

  const handleClick = (evt) => {
    evt.preventDefault();

    return onChange(!value);
  };

  return (
    <button className={`button button--unstyled ${sliderBtnClass}`} onClick={handleClick}>
      <div className={sliderDotClass} />
    </button>
  );
};

Slider.propTypes = {
  onChange: PropTypes.func,
  value: PropTypes.bool,
};

export default Slider;

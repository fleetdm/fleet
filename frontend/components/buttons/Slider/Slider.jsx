import React, { PropTypes } from 'react';
import classnames from 'classnames';

const Slider = ({ onClick, engaged }) => {
  const baseClass = 'slider-wrap';

  const sliderBtnClass = classnames(
    baseClass,
    { [`${baseClass}--active`]: engaged }
  );

  const sliderDotClass = classnames(
    `${baseClass}__dot`,
    { [`${baseClass}__dot--active`]: engaged }
  );

  return (
    <button className={`button button--unstyled ${sliderBtnClass}`} onClick={onClick}>
      <div className={sliderDotClass} />
    </button>
  );
};

Slider.propTypes = {
  engaged: PropTypes.bool,
  onClick: PropTypes.func,
};

export default Slider;

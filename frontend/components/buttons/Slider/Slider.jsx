import React, { PropTypes } from 'react';
import radium from 'radium';

import componentStyles from './styles';

const Slider = ({ onClick, engaged }) => {
  const { containerStyles, buttonStyles } = componentStyles;

  return (
    <button className="btn--unstyled" onClick={onClick} style={containerStyles(engaged)}>
      <div style={buttonStyles(engaged)} />
    </button>
  );
};

Slider.propTypes = {
  engaged: PropTypes.bool,
  onClick: PropTypes.func,
};

export default radium(Slider);

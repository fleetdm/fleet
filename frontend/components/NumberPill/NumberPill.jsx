import React, { PropTypes } from 'react';

const NumberPill = ({ number }) => {
  return <span className="number-pill">{number}</span>;
};

NumberPill.propTypes = {
  number: PropTypes.number,
};

export default NumberPill;

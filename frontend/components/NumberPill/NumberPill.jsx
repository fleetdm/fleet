import React from "react";
import PropTypes from "prop-types";

const NumberPill = ({ number }) => {
  return <span className="number-pill">{number}</span>;
};

NumberPill.propTypes = {
  number: PropTypes.number,
};

export default NumberPill;

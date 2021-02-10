import React from 'react';
import PropTypes from 'prop-types';

const TextCell = (props) => {
  const {
    value,
    formatter = val => val, // identity function if no formatter is provided
  } = props;

  return (
    <span>
      {formatter(value)}
    </span>
  );
};

TextCell.propTypes = {
  value: PropTypes.oneOfType([
    PropTypes.string,
    PropTypes.number,
  ]),
  formatter: PropTypes.func,
};

export default TextCell;

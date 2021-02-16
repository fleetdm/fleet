import React from 'react';
import PropTypes from 'prop-types';

const HeaderCell = (props) => {
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

HeaderCell.propTypes = {
  value: PropTypes.oneOfType([
    PropTypes.string,
    PropTypes.number,
  ]),
  formatter: PropTypes.func,
};

export default HeaderCell;

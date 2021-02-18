import React from 'react';
import PropTypes from 'prop-types';

// TODO: consider removing this and implementing css in HeaderCell.
const SortArrow = (props) => {
  const { isSortedDesc } = props;
  if (isSortedDesc === undefined) return null;

  return isSortedDesc
    ? <span>desc</span>
    : <span>asc</span>;
};

SortArrow.propTypes = {
  isSortedDesc: PropTypes.bool,
}

const HeaderCell = (props) => {
  const {
    value,
    isSortedDesc,
  } = props;

  return (
    <div>
      <span>{value}</span>
      <SortArrow isSortedDesc={isSortedDesc} />
    </div>
  );
};

HeaderCell.propTypes = {
  value: PropTypes.string,
  isSortedDesc: PropTypes.bool,
};

export default HeaderCell;

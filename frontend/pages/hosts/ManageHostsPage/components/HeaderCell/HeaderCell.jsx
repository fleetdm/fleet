import React from 'react';
import PropTypes from 'prop-types';

const HeaderCell = (props) => {
  const {
    value,
    isSortedDesc,
  } = props;

  let sortArrowClass = '';
  if (isSortedDesc === undefined) {
    sortArrowClass = '';
  } else if (isSortedDesc) {
    sortArrowClass = 'descending';
  } else {
    sortArrowClass = 'ascending';
  }

  return (
    <div className={`header-cell ${sortArrowClass}`}>
      <span>{value}</span>
      <div className="sort-arrows">
        <span className="ascending-arrow" />
        <span className="descending-arrow" />
      </div>
    </div>
  );
};

HeaderCell.propTypes = {
  value: PropTypes.string,
  isSortedDesc: PropTypes.bool,
};

export default HeaderCell;

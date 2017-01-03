import React, { PropTypes } from 'react';

const ClickableTableRow = ({ children, className, onClick }) => {
  /* eslint-disable jsx-a11y/no-static-element-interactions */
  return <tr className={className} onClick={onClick} tabIndex={-1}>{children}</tr>;
  /* eslint-enable jsx-a11y/no-static-element-interactions */
};

ClickableTableRow.propTypes = {
  children: PropTypes.node,
  className: PropTypes.string,
  onClick: PropTypes.func.isRequired,
};

export default ClickableTableRow;

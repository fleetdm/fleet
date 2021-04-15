import React from "react";
import PropTypes from "prop-types";

const ClickableTableRow = ({ children, className, onClick, onDoubleClick }) => {
  /* eslint-disable jsx-a11y/no-static-element-interactions */
  return (
    <tr
      className={className}
      onClick={onClick}
      onDoubleClick={onDoubleClick}
      tabIndex={-1}
    >
      {children}
    </tr>
  );
  /* eslint-enable jsx-a11y/no-static-element-interactions */
};

ClickableTableRow.propTypes = {
  children: PropTypes.node,
  className: PropTypes.string,
  onClick: PropTypes.func.isRequired,
  onDoubleClick: PropTypes.func,
};

export default ClickableTableRow;

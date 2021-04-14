import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

const baseClass = "warning-banner";

const WarningBanner = ({ children, className, shouldShowWarning }) => {
  if (!shouldShowWarning) {
    return null;
  }

  const fullClassName = classnames(baseClass, className);

  return (
    <div className={fullClassName}>
      <div className={`${baseClass}__message`}>{children}</div>
    </div>
  );
};

WarningBanner.propTypes = {
  children: PropTypes.node,
  className: PropTypes.string,
  shouldShowWarning: PropTypes.bool.isRequired,
};

export default WarningBanner;

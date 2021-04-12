import React from "react";
import PropTypes from "prop-types";
import classnames from "classnames";

import KolideIcon from "components/icons/KolideIcon";

const baseClass = "persistent-flash";

const PersistentFlash = ({ message }) => {
  const klass = classnames(baseClass, `${baseClass}--error`);

  return (
    <div className={klass}>
      <div className={`${baseClass}__content`}>
        <KolideIcon name="warning-filled" /> <span>{message}</span>
      </div>
    </div>
  );
};

PersistentFlash.propTypes = {
  message: PropTypes.string.isRequired,
};

export default PersistentFlash;

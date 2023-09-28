import React from "react";
import classnames from "classnames";

import DataError from "components/DataError";

const baseClass = "mdm-sso-error";

interface ISSOErrorProps {
  className?: string;
}

const SSOError = ({ className }: ISSOErrorProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <DataError className={classNames}>
      <p>
        Select <strong>Cancel</strong> and try again. If this keeps happening,
        please contact IT support.
      </p>
    </DataError>
  );
};

export default SSOError;

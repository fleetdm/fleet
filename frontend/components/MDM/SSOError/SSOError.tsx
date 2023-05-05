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
      <p>Please contact your IT admin at +1-(415)-651-2575.</p>
    </DataError>
  );
};

export default SSOError;

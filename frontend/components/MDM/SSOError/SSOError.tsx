import React from "react";
import classnames from "classnames";

import DataError from "components/DataError";

const baseClass = "mdm-sso-error";

interface ISSOErrorProps {
  className?: string;
  /** The sign-in took longer than the SSO session window, so there's nothing
   * left to verify against. Retrying works, which the generic copy doesn't say. */
  sessionExpired?: boolean;
}

const SSOError = ({ className, sessionExpired = false }: ISSOErrorProps) => {
  const classNames = classnames(baseClass, className);

  return (
    <DataError className={classNames}>
      {sessionExpired ? (
        <p>
          Your session may have timed out. Please exit and try again. Contact
          your IT support if the error persists.
        </p>
      ) : (
        <p>
          Please try again. If this keeps happening, please contact IT support.
        </p>
      )}
    </DataError>
  );
};

export default SSOError;

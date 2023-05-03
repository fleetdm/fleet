import React from "react";

import DataError from "components/DataError";

const baseClass = "mdm-sso-error";

const SSOError = () => {
  return (
    <DataError className={baseClass}>
      <p>Please contact your IT admin at +1-(415)-651-2575.</p>
    </DataError>
  );
};

export default SSOError;

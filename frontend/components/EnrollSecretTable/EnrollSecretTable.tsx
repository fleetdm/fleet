import React, { Component, useState } from "react";
import PropTypes from "prop-types";

import { IEnrollSecret } from "interfaces/enroll_secret";
import EnrollSecretRow from "./EnrollSecretRow";

const baseClass = "enroll-secrets";

interface IEnrollSecretRowProps {
  secrets: IEnrollSecret[] | undefined;
  toggleSecretEditorModal?: () => void;
  toggleDeleteSecretModal?: () => void;
}
const EnrollSecretTable = ({
  secrets,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
}: IEnrollSecretRowProps): JSX.Element | null => {
  let enrollSecretsClass = baseClass;
  if (!secrets) {
    return null;
  }

  if (secrets.length === 0) {
    return (
      <div className={baseClass}>
        <em>No active enroll secrets.</em>
      </div>
    );
  } else if (secrets.length > 1)
    enrollSecretsClass += ` ${baseClass}--multiple-secrets`;

  if (toggleSecretEditorModal && toggleDeleteSecretModal) {
    return (
      <div className={enrollSecretsClass}>
        {secrets.map(({ secret }) => (
          <EnrollSecretRow
            secret={secret}
            key={secret}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
          />
        ))}
      </div>
    );
  }
  return (
    <div className={enrollSecretsClass}>
      {secrets.map(({ secret }) => (
        <EnrollSecretRow secret={secret} />
      ))}
    </div>
  );
};

export default EnrollSecretTable;
export { EnrollSecretRow };

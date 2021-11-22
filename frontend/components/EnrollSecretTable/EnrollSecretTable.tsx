import React from "react";

import { IEnrollSecret } from "interfaces/enroll_secret";
import EnrollSecretRow from "./EnrollSecretRow";

const baseClass = "enroll-secrets";

interface IEnrollSecretRowProps {
  secrets: IEnrollSecret[] | undefined;
  toggleSecretEditorModal?: () => void;
  toggleDeleteSecretModal?: () => void;
  setSelectedSecret: React.Dispatch<
    React.SetStateAction<IEnrollSecret | undefined>
  >;
}
const EnrollSecretTable = ({
  secrets,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
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
        {secrets.map((secretInfo) => (
          <EnrollSecretRow
            secret={secretInfo}
            key={secretInfo.secret}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            setSelectedSecret={setSelectedSecret}
          />
        ))}
      </div>
    );
  }
  return (
    <div className={enrollSecretsClass}>
      {secrets.map((secretInfo) => (
        <EnrollSecretRow secret={secretInfo} key={secretInfo.secret} />
      ))}
    </div>
  );
};

export default EnrollSecretTable;
export { EnrollSecretRow };

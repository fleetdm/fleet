import React from "react";

import { IEnrollSecret } from "interfaces/enroll_secret";
import EnrollSecretRow from "./EnrollSecretRow";

const baseClass = "enroll-secrets";

interface IEnrollSecretRowProps {
  secrets: IEnrollSecret[] | undefined;
  toggleSecretEditorModal?: () => void;
  toggleDeleteSecretModal?: () => void;
  setSelectedSecret?: React.Dispatch<
    React.SetStateAction<IEnrollSecret | undefined>
  >;
}
const EnrollSecretTable = ({
  secrets,
  toggleSecretEditorModal,
  toggleDeleteSecretModal,
  setSelectedSecret,
}: IEnrollSecretRowProps): JSX.Element | null => {
  if (!secrets) {
    return null;
  }

  if (secrets.length === 0) {
    return (
      <div className={baseClass}>
        <em>No active enroll secrets.</em>
      </div>
    );
  }

  if (toggleSecretEditorModal && toggleDeleteSecretModal) {
    return (
      <>
        {secrets.map((secretInfo) => (
          <EnrollSecretRow
            secret={secretInfo}
            key={secretInfo.secret}
            toggleSecretEditorModal={toggleSecretEditorModal}
            toggleDeleteSecretModal={toggleDeleteSecretModal}
            setSelectedSecret={setSelectedSecret}
          />
        ))}
      </>
    );
  }

  return (
    <>
      {secrets.map((secretInfo) => {
        return <EnrollSecretRow secret={secretInfo} key={secretInfo.secret} />;
      })}
    </>
  );
};

export default EnrollSecretTable;
export { EnrollSecretRow };

import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
// @ts-ignore
import {
  IJiraIntegration,
  IJiraIntegrationIndexed,
} from "interfaces/integration";
import IntegrationForm from "../IntegrationForm";

const baseClass = "edit-team-modal";

interface IEditIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IJiraIntegration[]) => void;
  backendValidators: { [key: string]: string };
  integrations: IJiraIntegration[];
  integrationEditing?: IJiraIntegrationIndexed;
  testingConnection: boolean;
}

const EditIntegrationModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  integrations,
  integrationEditing,
  testingConnection,
}: IEditIntegrationModalProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  return (
    <Modal title={"Edit integration"} onExit={onCancel} className={baseClass}>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection to Jira</b>
          <Spinner />
        </div>
      ) : (
        <IntegrationForm
          onCancel={onCancel}
          onSubmit={onSubmit}
          integrations={integrations}
          integrationEditing={integrationEditing}
        />
      )}
    </Modal>
  );
};

export default EditIntegrationModal;

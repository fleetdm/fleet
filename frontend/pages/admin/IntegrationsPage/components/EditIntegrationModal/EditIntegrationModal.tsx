import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
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
}

const EditIntegrationModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  integrations,
  integrationEditing,
}: IEditIntegrationModalProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  return (
    <Modal title={"Edit integration"} onExit={onCancel} className={baseClass}>
      <IntegrationForm
        onCancel={onCancel}
        onSubmit={onSubmit}
        integrations={integrations}
        integrationEditing={integrationEditing}
      />
    </Modal>
  );
};

export default EditIntegrationModal;

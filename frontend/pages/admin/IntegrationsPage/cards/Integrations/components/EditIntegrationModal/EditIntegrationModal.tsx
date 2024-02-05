import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
import Spinner from "components/Spinner";
import {
  IIntegration,
  IIntegrations,
  IIntegrationTableData,
} from "interfaces/integration";
import IntegrationForm from "../IntegrationForm";

const baseClass = "edit-team-modal";

interface IEditIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IIntegration[]) => void;
  backendValidators: { [key: string]: string };
  integrations: IIntegrations;
  integrationEditing?: IIntegrationTableData;
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
    <Modal title="Edit integration" onExit={onCancel} className={baseClass}>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection</b>
          <Spinner />
        </div>
      ) : (
        <>
          <p>
            <b>Ticket destination:</b>
            <br />
            {integrationEditing?.type === "jira" ? "Jira" : "Zendesk"}
          </p>
          <IntegrationForm
            onCancel={onCancel}
            onSubmit={onSubmit}
            integrations={integrations}
            integrationEditing={integrationEditing}
            integrationEditingUrl={integrationEditing?.url || ""}
            integrationEditingUsername={integrationEditing?.username || ""}
            integrationEditingEmail={integrationEditing?.email || ""}
            integrationEditingApiToken={integrationEditing?.apiToken || ""}
            integrationEditingProjectKey={integrationEditing?.projectKey || ""}
            integrationEditingGroupId={integrationEditing?.groupId || 0}
            integrationEnableSoftwareVulnerabilities={
              integrationEditing?.enableSoftwareVulnerabilities || false
            }
            integrationEditingType={integrationEditing?.type}
          />
        </>
      )}
    </Modal>
  );
};

export default EditIntegrationModal;

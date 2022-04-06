import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
import InfoBanner from "components/InfoBanner/InfoBanner";
// @ts-ignore
import FleetIcon from "components/icons/FleetIcon";
import Spinner from "components/Spinner";
import { IJiraIntegration } from "interfaces/integration";
import IntegrationForm from "../IntegrationForm";

const baseClass = "create-integration-modal";

interface ICreateIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (jiraIntegrationSubmitData: IJiraIntegration[]) => void;
  serverErrors?: { base: string; email: string };
  backendValidators: { [key: string]: string };
  integrations: IJiraIntegration[];
  testingConnection: boolean;
}

const CreateIntegrationModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  integrations,
  testingConnection,
}: ICreateIntegrationModalProps): JSX.Element => {
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  return (
    <Modal title={"Add integration"} onExit={onCancel} className={baseClass}>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection to Jira</b>
          <Spinner />
        </div>
      ) : (
        <>
          <InfoBanner className={`${baseClass}__sandbox-info`}>
            <p className={`${baseClass}__info-header`}>
              Fleet supports Jira as a ticket destination.&nbsp;
              <a
                href="https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title="
                target="_blank"
                rel="noopener noreferrer"
              >
                Suggest a new destination&nbsp;
                <FleetIcon name="external-link" />
              </a>
            </p>
          </InfoBanner>
          <IntegrationForm
            onCancel={onCancel}
            onSubmit={onSubmit}
            integrations={integrations}
          />
        </>
      )}
    </Modal>
  );
};

export default CreateIntegrationModal;

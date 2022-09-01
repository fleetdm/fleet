import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/Spinner";
import { IIntegration, IIntegrations } from "interfaces/integration";
import IntegrationForm from "../IntegrationForm";
import ExternalURLIcon from "../../../../../../assets/images/icon-external-url-12x12@2x.png";

const baseClass = "create-integration-modal";

interface ICreateIntegrationModalProps {
  onCancel: () => void;
  onSubmit: (
    integrationSubmitData: IIntegration[],
    integrationDestination: string
  ) => void;
  serverErrors?: { base: string; email: string };
  backendValidators: { [key: string]: string };
  integrations: IIntegrations;
  testingConnection: boolean;
}

const destinationOptions = [
  { label: "Jira", value: "jira" },
  { label: "Zendesk", value: "zendesk" },
];

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
  const [destination, setDestination] = useState("jira");

  const onDestinationChange = (value: string) => {
    setDestination(value);
  };

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  return (
    <Modal title={"Add integration"} onExit={onCancel} className={baseClass}>
      {testingConnection ? (
        <div className={`${baseClass}__testing-connection`}>
          <b>Testing connection</b>
          <Spinner />
        </div>
      ) : (
        <>
          <div className={`${baseClass}__info-header`}>
            <Dropdown
              label="Ticket destination"
              name="destination"
              onChange={onDestinationChange}
              value={destination}
              options={destinationOptions}
              classname={`${baseClass}__destination-dropdown`}
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
            />
            <a
              href="https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title="
              target="_blank"
              rel="noopener noreferrer"
            >
              Suggest a new destination&nbsp;
              <img alt="Open external link" src={ExternalURLIcon} />
            </a>
          </div>
          <IntegrationForm
            onCancel={onCancel}
            onSubmit={onSubmit}
            integrations={integrations}
            destination={destination}
          />
        </>
      )}
    </Modal>
  );
};

export default CreateIntegrationModal;

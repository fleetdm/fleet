import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import CustomLink from "components/CustomLink";
import { IIntegration, IIntegrations } from "interfaces/integration";
import IntegrationForm from "../IntegrationForm";

const baseClass = "add-integration-modal";

interface IAddIntegrationModalProps {
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

const AddIntegrationModal = ({
  onCancel,
  onSubmit,
  backendValidators,
  integrations,
  testingConnection,
}: IAddIntegrationModalProps): JSX.Element => {
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
    <Modal title="Add integration" onExit={onCancel} className={baseClass}>
      <div className="form">
        {!testingConnection && (
          <>
            <Dropdown
              label="Ticket destination"
              name="destination"
              onChange={onDestinationChange}
              value={destination}
              options={destinationOptions}
              classname={`${baseClass}__destination-dropdown`}
              wrapperClassName={`${baseClass}__form-field ${baseClass}__form-field--platform`}
            />
            <CustomLink
              url="https://github.com/fleetdm/fleet/issues/new?assignees=&labels=idea&template=feature-request.md&title="
              text="Suggest a new destination"
              newTab
            />
          </>
        )}
        <IntegrationForm
          onCancel={onCancel}
          onSubmit={onSubmit}
          integrations={integrations}
          destination={destination}
          testingConnection={testingConnection}
        />
      </div>
    </Modal>
  );
};

export default AddIntegrationModal;

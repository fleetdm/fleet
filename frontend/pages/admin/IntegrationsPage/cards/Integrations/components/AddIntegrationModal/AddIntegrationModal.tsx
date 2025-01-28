import React, { useState, useEffect } from "react";

import Modal from "components/Modal";
import { SingleValue } from "react-select-5";
import DropdownWrapper from "components/forms/fields/DropdownWrapper";
import { CustomOptionType } from "components/forms/fields/DropdownWrapper/DropdownWrapper";
import CustomLink from "components/CustomLink";
import { IIntegration, IZendeskJiraIntegrations } from "interfaces/integration";
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
  integrations: IZendeskJiraIntegrations;
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

  const onDestinationChange = (
    selectedDestination: SingleValue<CustomOptionType>
  ) => {
    setDestination(selectedDestination?.value || "jira");
  };

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  return (
    <Modal title="Add integration" onExit={onCancel} className={baseClass}>
      <div className="form">
        {!testingConnection && (
          <>
            <DropdownWrapper
              name="destination"
              label="Ticket destination"
              onChange={onDestinationChange}
              value={destination}
              options={destinationOptions}
              className={`${baseClass}__destination-dropdown`}
              wrapperClassname={`${baseClass}__form-field ${baseClass}__form-field--platform`}
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

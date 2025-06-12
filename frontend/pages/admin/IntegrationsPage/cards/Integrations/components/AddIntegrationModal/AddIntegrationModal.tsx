import React, { useState, useContext } from "react";

import { AppContext } from "context/app";

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
  integrations,
  testingConnection,
}: IAddIntegrationModalProps): JSX.Element => {
  const gitOpsModeEnabled = useContext(AppContext).config?.gitops
    .gitops_mode_enabled;

  const [destination, setDestination] = useState("jira");

  const onDestinationChange = (
    selectedDestination: SingleValue<CustomOptionType>
  ) => {
    setDestination(selectedDestination?.value || "jira");
  };

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
              isDisabled={gitOpsModeEnabled}
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
          gitOpsModeEnabled={gitOpsModeEnabled}
        />
      </div>
    </Modal>
  );
};

export default AddIntegrationModal;

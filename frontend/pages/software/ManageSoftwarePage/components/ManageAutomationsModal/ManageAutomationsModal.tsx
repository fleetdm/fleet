import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";

import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import { useDeepEffect } from "utilities/hooks";
import { size } from "lodash";

import PreviewPayloadModal from "../PreviewPayloadModal";

interface IManageAutomationsModalProps {
  onCancel: () => void;
  onCreateWebhookSubmit: (formData: IWebhookSoftwareVulnerabilities) => void;
  togglePreviewPayloadModal: () => void;
  showPreviewPayloadModal: boolean;
  softwareVulnerabilityWebhookEnabled?: boolean;
  currentDestinationUrl?: string;
}

interface ICheckedSoftwareAutomation {
  name?: string;
  accessor: string;
  isChecked?: boolean;
}

const useCheckboxListStateManagement = (
  softwareVulnerabilityWebhookEnabled = false
) => {
  // Ability to add future software automations
  const availableSoftwareAutomations: ICheckedSoftwareAutomation[] = [
    { accessor: "vulnerability", name: "Enable vulnerability automations" },
  ];
  const currentSoftwareAutomations: ICheckedSoftwareAutomation[] = softwareVulnerabilityWebhookEnabled
    ? availableSoftwareAutomations
    : [];

  const [softwareAutomationsItems, setSoftwareAutomationsItems] = useState<
    ICheckedSoftwareAutomation[]
  >(() => {
    return (
      availableSoftwareAutomations &&
      availableSoftwareAutomations.map(
        (automation: ICheckedSoftwareAutomation) => {
          return {
            name: automation.name,
            accessor: automation.accessor,
            isChecked: currentSoftwareAutomations.some(
              (currentSoftwareAutomationItem: ICheckedSoftwareAutomation) =>
                currentSoftwareAutomationItem.accessor === automation.accessor
            ),
          };
        }
      )
    );
  });

  const updateSoftwareAutomationsItems = (
    softwareAutomationAccessor: string
  ) => {
    setSoftwareAutomationsItems((prevState) => {
      const selectedSoftwareAutomation = softwareAutomationsItems.find(
        (softwareAutomationItem) =>
          softwareAutomationItem.accessor === softwareAutomationAccessor
      );

      const updatedSoftwareAutomation = selectedSoftwareAutomation && {
        ...selectedSoftwareAutomation,
        isChecked:
          !!selectedSoftwareAutomation && !selectedSoftwareAutomation.isChecked,
      };

      // this is replacing the softwareAutomation object with the updatedSoftwareAutomation we just created
      const newState = prevState.map((currentSoftwareAutomation) => {
        return currentSoftwareAutomation.accessor ===
          softwareAutomationAccessor && updatedSoftwareAutomation
          ? updatedSoftwareAutomation
          : currentSoftwareAutomation;
      });
      return newState;
    });
  };

  return {
    softwareAutomationsItems,
    updateSoftwareAutomationsItems,
  };
};

const validateWebhookURL = (url: string) => {
  const errors: { [key: string]: string } = {};

  if (url === "") {
    errors.url = "Please add a destination URL";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const baseClass = "manage-automations-modal";

const ManageAutomationsModal = ({
  onCancel: onReturnToApp,
  onCreateWebhookSubmit,
  togglePreviewPayloadModal,
  showPreviewPayloadModal,
  softwareVulnerabilityWebhookEnabled,
  currentDestinationUrl,
}: IManageAutomationsModalProps): JSX.Element => {
  const [destination_url, setDestinationUrl] = useState<string>(
    currentDestinationUrl || ""
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const {
    softwareAutomationsItems,
    updateSoftwareAutomationsItems,
  } = useCheckboxListStateManagement(softwareVulnerabilityWebhookEnabled);

  useDeepEffect(() => {
    if (destination_url) {
      setErrors({});
    }
  }, [destination_url]);

  const onURLChange = (value: string) => {
    setDestinationUrl(value);
  };

  const handleSaveAutomation = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Ability to add future software automations
    const vulnerabilityWebhook = softwareAutomationsItems.find(
      (softwareAutomationItem) =>
        softwareAutomationItem.accessor === "vulnerability"
    );

    const { valid, errors: newErrors } = validateWebhookURL(destination_url);
    setErrors({
      ...errors,
      ...newErrors,
    });

    // URL validation only needed if software automation is checked
    if (valid || !vulnerabilityWebhook?.isChecked) {
      onCreateWebhookSubmit({
        destination_url,
        enable_vulnerabilities_webhook: vulnerabilityWebhook?.isChecked,
      });

      onReturnToApp();
    }
  };

  if (showPreviewPayloadModal) {
    return <PreviewPayloadModal onCancel={togglePreviewPayloadModal} />;
  }

  return (
    <Modal
      onExit={onReturnToApp}
      title={"Manage automations"}
      className={baseClass}
    >
      <div className={baseClass}>
        <div className={`${baseClass}__software-select-items`}>
          {softwareAutomationsItems &&
            softwareAutomationsItems.map(
              (softwareItem: ICheckedSoftwareAutomation) => {
                const { isChecked, name, accessor } = softwareItem;
                return (
                  <div key={accessor} className={`${baseClass}__team-item`}>
                    <Checkbox
                      value={isChecked}
                      name={name}
                      onChange={() =>
                        updateSoftwareAutomationsItems(softwareItem.accessor)
                      }
                    >
                      {name}
                    </Checkbox>
                  </div>
                );
              }
            )}
        </div>
        <div className={`${baseClass}__software-automation-description`}>
          <p>
            A request will be sent to your configured <b>Destination URL</b> if
            a detected vulnerability (CVE) was published in the last 2 days.
          </p>
        </div>
        <div className="tooltip-wrap tooltip-wrap--input">
          <InputField
            inputWrapperClass={`${baseClass}__url-input`}
            name="webhook-url"
            label={"Destination URL"}
            type={"text"}
            value={destination_url}
            onChange={onURLChange}
            error={errors.url}
            hint={
              "For each new vulnerability detected, Fleet will send a JSON payload to this URL with a list of the affected hosts."
            }
            placeholder={"https://server.com/example"}
            tooltip="Provide a URL to deliver a webhook request to."
          />
        </div>
        <Button
          type="button"
          variant="text-link"
          onClick={togglePreviewPayloadModal}
        >
          Preview payload
        </Button>

        <div className={`${baseClass}__button-wrap`}>
          <Button
            className={`${baseClass}__btn`}
            onClick={onReturnToApp}
            variant="inverse"
          >
            Cancel
          </Button>
          <Button
            className={`${baseClass}__btn`}
            type="submit"
            variant="brand"
            onClick={handleSaveAutomation}
          >
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
};

export default ManageAutomationsModal;

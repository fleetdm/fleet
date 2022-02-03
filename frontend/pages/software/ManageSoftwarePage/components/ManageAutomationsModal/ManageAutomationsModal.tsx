import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import IconToolTip from "components/IconToolTip";
import validURL from "components/forms/validators/valid_url";

import { IWebhookSoftwareVulnerabilities } from "interfaces/webhook";
import { useDeepEffect } from "utilities/hooks";
import { size } from "lodash";

import PreviewPayloadModal from "../PreviewPayloadModal";
import { createSecureContext } from "tls";

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
      availableSoftwareAutomations.map((automation: any) => {
        return {
          name: automation.name,
          accessor: automation.accessor,
          isChecked: currentSoftwareAutomations.some(
            (currentSoftwareAutomationItem: ICheckedSoftwareAutomation) =>
              currentSoftwareAutomationItem.accessor === automation.accessor
          ),
        };
      })
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

  if (!validURL(url)) {
    errors.url = "Please add a valid destination URL";
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

    const { valid, errors: newErrors } = validateWebhookURL(destination_url);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      // Ability to add future software automations
      const vulnerabilityWebhook = softwareAutomationsItems.find(
        (softwareAutomationItem) =>
          softwareAutomationItem.accessor === "vulnerability"
      );

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
          {softwareAutomationsItems && // Allows for more software automations to be set in the future
            softwareAutomationsItems.map((softwareItem: any) => {
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
            })}
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
          />
          <IconToolTip
            isHtml
            text={"<p>Provide a URL to deliver a<br />webhook request to.</p>"}
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

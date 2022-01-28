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

interface IManageAutomationsModalProps {
  onCancel: () => void;
  onCreateWebhookSubmit: (formData: IWebhookSoftwareVulnerabilities) => void;
  togglePreviewPayloadModal: () => void;
  showPreviewPayloadModal: boolean;
  availableSoftwareAutomations: any;
  currentSoftwareAutomations?: number[];
  currentDestinationUrl?: string;
}

interface ICheckedPolicy {
  name?: string;
  id: number;
  isChecked: boolean;
}

const useCheckboxListStateManagement = (
  availableSoftwareAutomations: any,
  currentSoftwareAutomations: number[] | undefined
) => {
  const [softwareAutomationsItems, setSoftwareAutomationsItems] = useState<
    ICheckedPolicy[]
  >(() => {
    return (
      availableSoftwareAutomations &&
      availableSoftwareAutomations.map((automation: any) => {
        return {
          name: automation.name,
          id: automation.id,
          isChecked:
            !!currentSoftwareAutomations &&
            currentSoftwareAutomations.includes(automation.id),
        };
      })
    );
  });

  const updateSoftwareAutomationsItems = (softwareAutomationId: number) => {
    setSoftwareAutomationsItems((prevState) => {
      const selectedSoftwareAutomation = softwareAutomationsItems.find(
        (softwareAutomationItem) =>
          softwareAutomationItem.id === softwareAutomationId
      );

      const updatedSoftwareAutomation = selectedSoftwareAutomation && {
        ...selectedSoftwareAutomation,
        isChecked:
          !!selectedSoftwareAutomation && !selectedSoftwareAutomation.isChecked,
      };

      // this is replacing the policy object with the updatedPolicy we just created.
      const newState = prevState.map((currentSoftwareAutomation) => {
        return currentSoftwareAutomation.id === softwareAutomationId &&
          updatedSoftwareAutomation
          ? updatedSoftwareAutomation
          : currentSoftwareAutomation;
      });
      return newState;
    });
  };

  return {
    softwareAutomationsItems: softwareAutomationsItems,
    updateSoftwareAutomationsItems: updateSoftwareAutomationsItems,
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
  availableSoftwareAutomations, // TODO: pass ManageAutomationsModal availableSoftwareAutomations
  currentSoftwareAutomations, // TODO: pass ManageAutomationsModal currentSoftwareAutomations
  currentDestinationUrl,
}: IManageAutomationsModalProps): JSX.Element => {
  const [destination_url, setDestinationUrl] = useState<string>(
    currentDestinationUrl || ""
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const {
    softwareAutomationsItems, // TODO: update for software automations
    updateSoftwareAutomationsItems, // TODO: update for software automations
  } = useCheckboxListStateManagement(
    availableSoftwareAutomations,
    currentSoftwareAutomations
  );

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
      const enable_vulnerabilities_webhook = true; // Leave nearest component in case we decide to add disabling as a UI feature

      onCreateWebhookSubmit({
        destination_url,
        enable_vulnerabilities_webhook,
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
              const { isChecked, name, id } = softwareItem;
              return (
                <div key={id} className={`${baseClass}__team-item`}>
                  <Checkbox
                    value={isChecked}
                    name={name}
                    onChange={() =>
                      updateSoftwareAutomationsItems(softwareItem.id)
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

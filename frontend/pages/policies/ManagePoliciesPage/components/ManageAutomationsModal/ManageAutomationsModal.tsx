import React, { useState } from "react";

import { IPolicy } from "interfaces/policy";
import { IWebhookFailingPolicies } from "interfaces/webhook";
import { useDeepEffect } from "utilities/hooks";
import { size } from "lodash";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import PreviewPayloadModal from "../PreviewPayloadModal";

interface IManageAutomationsModalProps {
  onCancel: () => void;
  onCreateWebhookSubmit: (formData: IWebhookFailingPolicies) => void;
  togglePreviewPayloadModal: () => void;
  showPreviewPayloadModal: boolean;
  availablePolicies: IPolicy[];
  currentAutomatedPolicies?: number[];
  currentDestinationUrl?: string;
}

interface ICheckedPolicy {
  name?: string;
  id: number;
  isChecked: boolean;
}

const useCheckboxListStateManagement = (
  allPolicies: IPolicy[],
  automatedPolicies: number[] | undefined
) => {
  const [policyItems, setPolicyItems] = useState<ICheckedPolicy[]>(() => {
    return (
      allPolicies &&
      allPolicies.map((policy) => {
        return {
          name: policy.name,
          id: policy.id,
          isChecked:
            !!automatedPolicies && automatedPolicies.includes(policy.id),
        };
      })
    );
  });

  const updatePolicyItems = (policyId: number) => {
    setPolicyItems((prevState) => {
      const selectedPolicy = policyItems.find(
        (policy) => policy.id === policyId
      );

      const updatedPolicy = selectedPolicy && {
        ...selectedPolicy,
        isChecked: !!selectedPolicy && !selectedPolicy.isChecked,
      };

      // this is replacing the policy object with the updatedPolicy we just created.
      const newState = prevState.map((currentPolicy) => {
        return currentPolicy.id === policyId && updatedPolicy
          ? updatedPolicy
          : currentPolicy;
      });
      return newState;
    });
  };

  return { policyItems, updatePolicyItems };
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
  availablePolicies,
  currentAutomatedPolicies,
  currentDestinationUrl,
}: IManageAutomationsModalProps): JSX.Element => {
  const [destination_url, setDestinationUrl] = useState<string>(
    currentDestinationUrl || ""
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});

  const { policyItems, updatePolicyItems } = useCheckboxListStateManagement(
    availablePolicies,
    currentAutomatedPolicies
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

    const policy_ids =
      policyItems &&
      policyItems
        .filter((policy) => policy.isChecked)
        .map((policy) => policy.id);
    const enable_failing_policies_webhook = true; // Leave nearest component in case we decide to add disabling as a UI feature

    // URL validation only needed if at least one policy is checked
    if (valid || policy_ids.length === 0) {
      onCreateWebhookSubmit({
        destination_url,
        policy_ids,
        enable_failing_policies_webhook,
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
        {availablePolicies && availablePolicies.length > 0 ? (
          <div className={`${baseClass}__policy-select-items`}>
            <p> Choose which policies you would like to listen to:</p>
            {policyItems &&
              policyItems.map((policyItem) => {
                const { isChecked, name, id } = policyItem;
                return (
                  <div key={id} className={`${baseClass}__team-item`}>
                    <Checkbox
                      value={isChecked}
                      name={name}
                      onChange={() => updatePolicyItems(policyItem.id)}
                    >
                      {name}
                    </Checkbox>
                  </div>
                );
              })}
          </div>
        ) : (
          <div className={`${baseClass}__no-policies`}>
            <b>You have no policies.</b>
            <p>Add a policy to turn on automations.</p>
          </div>
        )}
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
              'For each policy, Fleet will send a JSON payload to this URL with a list of the hosts that updated their answer to "No."'
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

import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import IconToolTip from "components/IconToolTip";

import { IPolicy, IPolicyFormData } from "interfaces/policy";
import { IAutomationFormData } from "interfaces/automation";
import { useDeepEffect } from "utilities/hooks";
import { size } from "lodash";

import PreviewPayloadModal from "../PreviewPayloadModal";

interface IPolicyCheckboxListItem extends IPolicy {
  isChecked: boolean | undefined;
}

interface IManageAutomationsModalProps {
  onCancel: () => void;
  onCreateAutomationsSubmit: (formData: IAutomationFormData) => void;
  togglePreviewPayloadModal: () => void;
  showPreviewPayloadModal: boolean;
  availablePolicies: IPolicy[];
  currentAutomatedPolicies: IPolicy[];
  onFormChange: (policies: IPolicyFormData[]) => void;
}

const validateAutomationURL = (url: string) => {
  const errors: { [key: string]: string } = {};

  if (!url) {
    errors.url = "URL name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const baseClass = "manage-automations-modal";

const ManageAutomationsModal = ({
  onCancel: onReturnToApp,
  onCreateAutomationsSubmit,
  togglePreviewPayloadModal,
  showPreviewPayloadModal,
  availablePolicies,
  currentAutomatedPolicies,
  onFormChange,
}: IManageAutomationsModalProps): JSX.Element => {
  const generateFormListItems = (
    allPolicies: IPolicy[],
    currentPolicies: IPolicy[]
  ): IPolicyCheckboxListItem[] => {
    return allPolicies.map((policy) => {
      const foundPolicy = currentPolicies.find(
        (currentPolicy) => currentPolicy.id === policy.id
      );
      return {
        ...policy,
        isChecked: foundPolicy !== undefined,
      };
    });
  };

  const useSelectedPolicyState = (
    allPolicies: IPolicy[],
    currentPolicies: IPolicy[],
    formChange: (policies: IPolicyFormData[]) => void
  ) => {
    const [policiesFormList, setPoliciesFormList] = useState(() => {
      return generateFormListItems(allPolicies, currentPolicies);
    });

    const updateSelectedPolicies = (
      policyId: number,
      newValue: any,
      updateType: string
    ) => {
      setPoliciesFormList((prevState: any) => {
        const updatedPolicyFormList = updateFormState(
          prevState,
          policyId,
          newValue,
          updateType
        );
        const selectedPoliciesData = generateSelectedPolicyData(
          updatedPolicyFormList
        );
        formChange(selectedPoliciesData);
        return updatedPolicyFormList;
      });
    };

    return [policiesFormList, updateSelectedPolicies] as const;
  };

  const [url, setURL] = useState<string>("");
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [policiesFormList, updateSelectedPolicies] = useSelectedPolicyState(
    availablePolicies,
    currentAutomatedPolicies,
    onFormChange
  );

  useDeepEffect(() => {
    if (url) {
      setErrors({});
    }
  }, [url]);

  const onURLChange = (value: string) => {
    setURL(value);
  };

  const handleSaveAutomation = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const { valid, errors: newErrors } = validateAutomationURL(url);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      onCreateAutomationsSubmit({ url });

      onReturnToApp;
    }
  };

  const generateSelectedPolicyData = (
    policiesFormList: IPolicyCheckboxListItem[]
  ): IPolicyFormData[] => {
    return policiesFormList.reduce(
      (selectedPolicies: IPolicyFormData[], policyItem) => {
        if (policyItem.isChecked) {
          selectedPolicies.push({
            id: policyItem.id,
            name: policyItem.name,
            description: policyItem.description,
            resolution: policyItem.resolution,
          });
        }
        return selectedPolicies;
      },
      []
    );
  };

  // handles the updating of the form items.
  // updates either selected state or the dropdown status of an item.
  const updateFormState = (
    prevPolicyItems: IPolicyCheckboxListItem[],
    policyId: number,
    newValue: any,
    updateType: string
  ): IPolicyCheckboxListItem[] => {
    const prevItemIndex = prevPolicyItems.findIndex(
      (item) => item.id === policyId
    );
    const prevItem = prevPolicyItems[prevItemIndex];
    prevItem.isChecked = newValue;

    return [...prevPolicyItems];
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
        <p> Choose which policy you would like to listen to:</p>
        <div className="tooltip-wrap tooltip-wrap--input">
          <InputField
            inputWrapperClass={`${baseClass}__url-input`}
            name="automations-url"
            label={"Destination URL"}
            type={"text"}
            value={url}
            onChange={onURLChange}
            error={errors.url}
            hint={
              'For each policy, Fleet will send a JSON payload to this URL with a list of the hosts that updated their answer to "No."'
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

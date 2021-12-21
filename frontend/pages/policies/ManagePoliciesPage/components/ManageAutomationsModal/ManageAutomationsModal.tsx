import React, { useState } from "react";

import Modal from "components/Modal";
import Button from "components/buttons/Button";
// @ts-ignore
import Checkbox from "components/forms/fields/Checkbox";
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
  currentAutomatedPolicies?: number[];
  currentDestinationUrl: string;
  onFormChange: (formData: IAutomationFormData) => void;
}

const validateAutomationURL = (url: string) => {
  const errors: { [key: string]: string } = {};

  if (!url) {
    errors.url = "URL name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

/* Handles all policies and returns all policies
with a boolean key isChecked based on current policies */
const generateFormListItems = (
  allPolicies: IPolicy[],
  currentAutomatedPolicies?: number[] | undefined
): IPolicyCheckboxListItem[] => {
  console.log("allPolicies", allPolicies);
  console.log("currentAutomatedPolicies", currentAutomatedPolicies);

  return allPolicies.map((policy) => {
    const foundPolicy =
      currentAutomatedPolicies?.find(
        (currentPolicy) => currentPolicy === policy.id
      ) || undefined;
    return {
      ...policy,
      isChecked: foundPolicy !== undefined,
    };
  });
};

/* Handles the generation of the form data eventually passed up to the parent
so we only want to send the selected policies. */
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

/* Handles the updating of the form items and updates the selected state.*/
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

/* TODO: What does this hook do? How/why does this work? */
const useSelectedPolicyState = (
  allPolicies: IPolicy[],
  currentAutomatedPolicies: number[],
  formChange: (policies: IPolicyFormData[]) => void
) => {
  const [policiesFormList, setPoliciesFormList] = useState(() => {
    return generateFormListItems(allPolicies, currentAutomatedPolicies);
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

const onSelectedPolicyChange = (policies: IPolicyFormData[]): void => {
  // TODO: rewrite
  // const { formData } = this.state;
  // this.setState({
  //   formData: {
  //     ...formData,
  //     policies,
  //   },
  // });
};

const baseClass = "manage-automations-modal";

const ManageAutomationsModal = ({
  onCancel: onReturnToApp,
  onCreateAutomationsSubmit,
  togglePreviewPayloadModal,
  showPreviewPayloadModal,
  availablePolicies,
  currentAutomatedPolicies,
  currentDestinationUrl,
  onFormChange,
}: IManageAutomationsModalProps): JSX.Element => {
  const [destination_url, setDestinationUrl] = useState<string>(
    currentDestinationUrl
  );
  const [errors, setErrors] = useState<{ [key: string]: string }>({});
  const [policiesFormList, updateSelectedPolicies] = useSelectedPolicyState(
    availablePolicies,
    currentAutomatedPolicies || [],
    onSelectedPolicyChange
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

    const { valid, errors: newErrors } = validateAutomationURL(destination_url);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (valid) {
      //TODO: FIX THIS
      // onCreateAutomationsSubmit({ destination_url, policiesFormList });

      onReturnToApp;
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
        <div className={`${baseClass}__policy-select-items`}>
          <p> Choose which policy you would like to listen to:</p>
          {policiesFormList.map((policyItem) => {
            const { isChecked, name, id } = policyItem;
            return (
              <div key={id} className={`${baseClass}__team-item`}>
                <Checkbox
                  value={isChecked}
                  name={name}
                  onChange={(newValue: boolean) =>
                    updateSelectedPolicies(policyItem.id, newValue, "checkbox")
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
            name="automations-url"
            label={"Destination URL"}
            type={"text"}
            value={destination_url}
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

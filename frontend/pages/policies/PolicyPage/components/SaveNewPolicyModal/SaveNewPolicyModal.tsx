import React, { useState, useContext, useEffect } from "react";
import { size } from "lodash";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPlatformSelector } from "hooks/usePlatformSelector";
import { IPolicyFormData } from "interfaces/policy";
import { SelectedPlatformString } from "interfaces/platform";
import useDeepEffect from "hooks/useDeepEffect";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import ReactTooltip from "react-tooltip";
import PremiumFeatureIconWithTooltip from "components/PremiumFeatureIconWithTooltip";
import { COLORS } from "styles/var/colors";

export interface ISaveNewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  setIsSaveNewPolicyModalOpen: (isOpen: boolean) => void;
  backendValidators: { [key: string]: string };
  platformSelector: IPlatformSelector;
  isUpdatingPolicy: boolean;
}

const validatePolicyName = (name: string) => {
  const errors: { [key: string]: string } = {};

  if (!name) {
    errors.name = "Policy name must be present";
  }

  const valid = !size(errors);
  return { valid, errors };
};

const SaveNewPolicyModal = ({
  baseClass,
  queryValue,
  onCreatePolicy,
  setIsSaveNewPolicyModalOpen,
  backendValidators,
  platformSelector,
  isUpdatingPolicy,
}: ISaveNewPolicyModalProps): JSX.Element => {
  const { isPremiumTier, isSandboxMode } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
    lastEditedQueryCritical,
    setLastEditedQueryPlatform,
  } = useContext(PolicyContext);

  const [name, setName] = useState(lastEditedQueryName);
  const [description, setDescription] = useState(lastEditedQueryDescription);
  const [resolution, setResolution] = useState(lastEditedQueryResolution);
  const [critical, setCritical] = useState(lastEditedQueryCritical);
  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  const disableSave = !platformSelector.isAnyPlatformSelected;

  useDeepEffect(() => {
    if (name) {
      setErrors({});
    }
  }, [name]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const handleSavePolicy = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const newPlatformString = platformSelector
      .getSelectedPlatforms()
      .join(",") as SelectedPlatformString;
    setLastEditedQueryPlatform(newPlatformString);

    const { valid: validName, errors: newErrors } = validatePolicyName(name);
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (!disableSave && validName) {
      onCreatePolicy({
        description,
        name,
        query: queryValue,
        resolution,
        platform: newPlatformString,
        critical,
      });
    }
  };

  return (
    <Modal
      title="Save policy"
      onExit={() => setIsSaveNewPolicyModalOpen(false)}
    >
      <>
        <form
          onSubmit={handleSavePolicy}
          className={`${baseClass}__save-modal-form`}
          autoComplete="off"
        >
          <InputField
            name="name"
            onChange={(value: string) => setName(value)}
            value={name}
            error={errors.name}
            inputClassName={`${baseClass}__policy-save-modal-name`}
            label="Name"
            helpText="What yes or no question does your policy ask about your hosts?"
            autofocus
            ignore1password
          />
          <InputField
            name="description"
            onChange={(value: string) => setDescription(value)}
            value={description}
            inputClassName={`${baseClass}__policy-save-modal-description`}
            label="Description"
            type="textarea"
          />
          <InputField
            name="resolution"
            onChange={(value: string) => setResolution(value)}
            value={resolution}
            inputClassName={`${baseClass}__policy-save-modal-resolution`}
            label="Resolution"
            type="textarea"
            helpText="What steps should an end user take to resolve a host that fails this policy? (optional)"
          />
          {platformSelector.render()}
          {isPremiumTier && (
            <div className="critical-checkbox-wrapper">
              {isSandboxMode && <PremiumFeatureIconWithTooltip />}
              <Checkbox
                name="critical-policy"
                onChange={(value: boolean) => setCritical(value)}
                value={critical}
                isLeftLabel
              >
                <TooltipWrapper
                  tipContent={
                    <p>
                      If automations are turned on, this
                      <br /> information is included.
                    </p>
                  }
                >
                  Critical:
                </TooltipWrapper>
              </Checkbox>
            </div>
          )}
          <div className="modal-cta-wrap">
            <span
              className={`${baseClass}__button-wrap--modal-save`}
              data-tip
              data-for={`${baseClass}__button--modal-save-tooltip`}
              data-tip-disable={!disableSave}
            >
              <Button
                type="submit"
                variant="brand"
                onClick={handleSavePolicy}
                disabled={disableSave}
                className="save-policy-loading"
                isLoading={isUpdatingPolicy}
              >
                Save policy
              </Button>
              <ReactTooltip
                className={`${baseClass}__button--modal-save-tooltip`}
                place="bottom"
                effect="solid"
                id={`${baseClass}__button--modal-save-tooltip`}
                backgroundColor={COLORS["tooltip-bg"]}
              >
                Select the platform(s) this
                <br />
                policy will be checked on
                <br />
                to save the policy.
              </ReactTooltip>
            </span>
            <Button
              className={`${baseClass}__button--modal-cancel`}
              onClick={() => setIsSaveNewPolicyModalOpen(false)}
              variant="inverse"
            >
              Cancel
            </Button>
          </div>
        </form>
      </>
    </Modal>
  );
};

export default SaveNewPolicyModal;

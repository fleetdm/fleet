import React, { useState, useContext, useEffect, useCallback } from "react";
import { size } from "lodash";
import classNames from "classnames";

import { AppContext } from "context/app";
import { PolicyContext } from "context/policy";
import { IPlatformSelector } from "hooks/usePlatformSelector";
import { IPolicyFormData } from "interfaces/policy";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import useDeepEffect from "hooks/useDeepEffect";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Checkbox from "components/forms/fields/Checkbox";
import TooltipWrapper from "components/TooltipWrapper";
import Button from "components/buttons/Button";
import Modal from "components/Modal";
import Icon from "components/Icon";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

export interface ISaveNewPolicyModalProps {
  baseClass: string;
  queryValue: string;
  onCreatePolicy: (formData: IPolicyFormData) => void;
  setIsSaveNewPolicyModalOpen: (isOpen: boolean) => void;
  backendValidators: { [key: string]: string };
  platformSelector: IPlatformSelector;
  isUpdatingPolicy: boolean;
  aiFeaturesDisabled?: boolean;
  isFetchingAutofillDescription: boolean;
  isFetchingAutofillResolution: boolean;
  onClickAutofillDescription: () => Promise<void>;
  onClickAutofillResolution: () => Promise<void>;
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
  aiFeaturesDisabled,
  isFetchingAutofillDescription,
  isFetchingAutofillResolution,
  onClickAutofillDescription,
  onClickAutofillResolution,
}: ISaveNewPolicyModalProps): JSX.Element => {
  const { isPremiumTier } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryResolution,
    lastEditedQueryCritical,
    setLastEditedQueryName,
    setLastEditedQueryPlatform,
    // TODO: Keep last edited query platform from resetting when cancelling out of modal and clicking save again
    setLastEditedQueryDescription,
    setLastEditedQueryResolution,
    setLastEditedQueryCritical,
  } = useContext(PolicyContext);

  const [errors, setErrors] = useState<{ [key: string]: string }>(
    backendValidators
  );

  const disableForm =
    isFetchingAutofillDescription || isFetchingAutofillResolution;
  const disableSave = !platformSelector.isAnyPlatformSelected || disableForm;

  useDeepEffect(() => {
    if (lastEditedQueryName) {
      setErrors({});
    }
  }, [lastEditedQueryName]);

  useEffect(() => {
    setErrors(backendValidators);
  }, [backendValidators]);

  const handleSavePolicy = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const newPlatformString = platformSelector
      .getSelectedPlatforms()
      .join(",") as CommaSeparatedPlatformString;
    setLastEditedQueryPlatform(newPlatformString);

    const { valid: validName, errors: newErrors } = validatePolicyName(
      lastEditedQueryName
    );
    setErrors({
      ...errors,
      ...newErrors,
    });

    if (!disableSave && validName) {
      onCreatePolicy({
        description: lastEditedQueryDescription,
        name: lastEditedQueryName,
        query: queryValue,
        resolution: lastEditedQueryResolution,
        platform: newPlatformString,
        critical: lastEditedQueryCritical,
      });
    }
  };

  const renderAutofillButton = useCallback(
    (labelName: "Description" | "Resolution") => {
      const isFetchingButton =
        (labelName === "Description" && isFetchingAutofillDescription) ||
        (labelName === "Resolution" && isFetchingAutofillResolution);

      return (
        <>
          <div
            data-tip
            data-for={`autofill-button-${labelName}`}
            // Tooltip shows except when fetching AI autofill
            data-tip-disable={disableForm}
            className="autofill-tooltip-wrapper"
          >
            <Button
              variant="text-icon"
              disabled={aiFeaturesDisabled || disableForm}
              onClick={
                labelName === "Description"
                  ? onClickAutofillDescription
                  : onClickAutofillResolution
              }
            >
              {isFetchingButton ? (
                "Thinking..."
              ) : (
                <>
                  <Icon name="sparkles" /> Autofill
                </>
              )}
            </Button>
          </div>
          <ReactTooltip
            className="autofill-button-tooltip"
            place="top"
            effect="solid"
            backgroundColor={COLORS["tooltip-bg"]}
            id={`autofill-button-${labelName}`}
            data-html
          >
            {aiFeaturesDisabled ? (
              "AI features are disabled in organization settings"
            ) : (
              <>
                Policy queries (SQL) will be sent to a <br />
                large language model (LLM). Fleet <br />
                doesn&apos;t use this data to train models.
              </>
            )}
          </ReactTooltip>
        </>
      );
    },
    [isFetchingAutofillDescription, isFetchingAutofillResolution, disableForm]
  );

  const renderAutofillLabel = useCallback(
    (labelName: "Description" | "Resolution") => {
      const labelClassName = classNames(`${baseClass}__autofill-label`, {
        [`${baseClass}__label--${labelName}`]: !!labelName,
      });

      return (
        <div className={labelClassName}>
          {labelName}
          {renderAutofillButton(labelName)}
        </div>
      );
    },
    [renderAutofillButton]
  );

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
            onChange={(value: string) => setLastEditedQueryName(value)}
            value={lastEditedQueryName}
            error={errors.name}
            inputClassName={`${baseClass}__policy-save-modal-name`}
            label="Name"
            autofocus
            ignore1password
            disabled={disableForm}
          />
          <InputField
            name="description"
            onChange={(value: string) => setLastEditedQueryDescription(value)}
            value={lastEditedQueryDescription}
            inputClassName={`${baseClass}__policy-save-modal-description`}
            label={renderAutofillLabel("Description")}
            helpText="How does this policy's failure put the organization at risk?"
            type="textarea"
            disabled={disableForm}
          />
          <InputField
            name="resolution"
            onChange={(value: string) => setLastEditedQueryResolution(value)}
            value={lastEditedQueryResolution}
            inputClassName={`${baseClass}__policy-save-modal-resolution`}
            label={renderAutofillLabel("Resolution")}
            type="textarea"
            helpText="If this policy fails, what should the end user expect?"
            disabled={disableForm}
          />
          {platformSelector.render()}
          {isPremiumTier && (
            <div className="critical-checkbox-wrapper">
              <Checkbox
                name="critical-policy"
                onChange={(value: boolean) => setLastEditedQueryCritical(value)}
                value={lastEditedQueryCritical}
                isLeftLabel
                disabled={disableForm}
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

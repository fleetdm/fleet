import React, { useState } from "react";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";

import { ICustomSCEPFormValidation, validateFormData } from "./helpers";

const baseClass = "ndes-form";

export interface ICustomSCEPFormData {
  name: string;
  scepURL: string;
  challenge: string;
}

interface ICustomSCEPFormProps {
  formData: ICustomSCEPFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const CustomSCEPForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  onChange,
  onSubmit,
  onCancel,
}: ICustomSCEPFormProps) => {
  const [
    formValidation,
    setFormValidation,
  ] = useState<ICustomSCEPFormValidation>({
    isValid: false,
  });

  const { name, scepURL, challenge } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  const onInputChange = (update: { name: string; value: string }) => {
    setFormValidation(
      validateFormData({ ...formData, [update.name]: update.value })
    );
    onChange(update);
  };

  return (
    <form onSubmit={onSubmitForm}>
      <div className={`${baseClass}__fields`}>
        <InputField
          label="Name"
          name="name"
          value={name}
          onChange={onInputChange}
          parseTarget
          placeholder="SCEP_WIFI"
          helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_SCEP_WIFI)."
        />
        <InputField
          label="SCEP URL"
          name="scepURL"
          value={scepURL}
          onChange={onInputChange}
          parseTarget
          placeholder="https://example.com/scep"
        />
        <InputField
          label="Challenge"
          name="challenge"
          value={challenge}
          onChange={onInputChange}
          parseTarget
          placeholder="••••••••••••"
          helpText="Password to authenticate with a SCEP server."
        />
      </div>
      <div className={`${baseClass}__cta`}>
        <TooltipWrapper
          tipContent="Complete all required fields to save."
          underline={false}
          position="top"
          disableTooltip={formValidation.isValid}
          showArrow
        >
          <Button
            type="submit"
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting}
          >
            {submitBtnText}
          </Button>
        </TooltipWrapper>
        <Button variant="inverse" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  );
};

export default CustomSCEPForm;

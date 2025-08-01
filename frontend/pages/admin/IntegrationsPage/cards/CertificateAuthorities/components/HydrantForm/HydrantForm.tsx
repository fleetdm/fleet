import React, { useContext, useMemo, useState } from "react";

import { AppContext } from "context/app";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button";
import TooltipWrapper from "components/TooltipWrapper";
import {
  validateFormData,
  IHydrantFormValidation,
  generateFormValidations,
} from "./helpers";

const baseClass = "hydrant-form";

export interface IHydrantFormData {
  name: string;
  url: string;
  clientId: string;
  clientSecret: string;
}

interface IHydrantFormProps {
  formData: IHydrantFormData;
  submitBtnText: string;
  isSubmitting: boolean;
  isEditing?: boolean;
  onChange: (update: { name: string; value: string }) => void;
  onSubmit: () => void;
  onCancel: () => void;
}

const HydrantForm = ({
  formData,
  submitBtnText,
  isSubmitting,
  isEditing = false,
  onChange,
  onSubmit,
  onCancel,
}: IHydrantFormProps) => {
  const { config } = useContext(AppContext);
  const validations = useMemo(
    () =>
      generateFormValidations(config?.integrations.digicert ?? [], isEditing),
    [config?.integrations.digicert, isEditing]
  );

  const [formValidation, setFormValidation] = useState<IHydrantFormValidation>(
    () => validateFormData(formData, validations)
  );

  const { name, url, clientId, clientSecret } = formData;

  const onSubmitForm = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    onSubmit();
  };

  const onInputChange = (update: { name: string; value: string }) => {
    setFormValidation(
      validateFormData(
        { ...formData, [update.name]: update.value },
        validations
      )
    );
    onChange(update);
  };

  return (
    <form className={baseClass} onSubmit={onSubmitForm}>
      <div className={`${baseClass}__fields`}>
        <InputField
          name="name"
          label="Name"
          value={name}
          onChange={onInputChange}
          error={formValidation.name?.message}
          helpText="Letters, numbers, and underscores only. Fleet will create configuration profile variables with the name as suffix (e.g. $FLEET_VAR_HYDRANT_DATA_WIFI_CERTIFICATE)."
          parseTarget
          placeholder="WIFI_CERTIFICATE"
        />
        <InputField
          name="url"
          label="URL"
          value={url}
          onChange={onInputChange}
          error={formValidation.url?.message}
          parseTarget
          helpText="EST endpoint provided by Hydrant."
          placeholder="https://example.hydrantid.com/.well-known/est/abc123"
        />
        <InputField
          name="clientId"
          label="Client ID"
          value={clientId}
          onChange={onInputChange}
          parseTarget
          helpText="Client ID provided by Hydrant."
        />
        <InputField
          name="clientSecret"
          label="Client secret"
          value={clientSecret}
          onChange={onInputChange}
          parseTarget
          helpText="Client secret provided by Hydrant."
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
            isLoading={isSubmitting}
            disabled={!formValidation.isValid || isSubmitting}
            type="submit"
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

export default HydrantForm;

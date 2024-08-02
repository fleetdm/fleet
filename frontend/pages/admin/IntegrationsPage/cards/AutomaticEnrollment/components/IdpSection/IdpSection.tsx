import React, { useCallback, useContext, useState } from "react";

import configAPI from "services/entities/config";

// @ts-ignore
import InputField from "components/forms/fields/InputField";
import CustomLink from "components/CustomLink/CustomLink";
import Button from "components/buttons/Button/Button";
import SectionHeader from "components/SectionHeader";
import validateUrl from "components/forms/validators/valid_url";
import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";
import { expandErrorReasonRequired } from "interfaces/errors";
import { AxiosResponse } from "axios";
import TooltipWrapper from "components/TooltipWrapper";

const baseClass = "idp-section";

type IIdpFormData = {
  idpName: string;
  entityId: string;
  metadataUrl?: string;
  metadata?: string;
};

type FormName = keyof IIdpFormData;

// TODO: backend is not validating all of these rules AFAICT

const ERROR_CONFIGS = {
  idpName: {
    isValid: (data: IIdpFormData) => data.idpName !== "",
    message: "Identity provider name is required.",
  },
  entityId: {
    isValid: (data: IIdpFormData) => data.entityId.length >= 5,
    message: "Entity ID must be 5 or more characters.",
  },
  metadataUrl: {
    isValid: (data: IIdpFormData) =>
      !data.metadataUrl || validateUrl({ url: data.metadataUrl }),
    message: "Metadata URL must be a valid URL.",
  },
  metadataOrMetadataUrl: {
    isValid: (data: IIdpFormData) => !!data.metadata || !!data.metadataUrl,
    message: "Metadata or Metadata URL is required.",
  },
} as const;

type FormError = keyof typeof ERROR_CONFIGS;

type FormErrors = Partial<Record<FormError, string>>;

const isEmptyFormData = (data: IIdpFormData) => {
  const values = Object.values(data);
  return !values.length || values.every((v) => v === "");
};

const validateForm = (data: IIdpFormData): FormErrors | null => {
  let formErrors: FormErrors | null = null;
  if (isEmptyFormData(data)) {
    // TODO: confirm whether we want to allow user to save an empty form or if should be treated
    // as a form error (what happens is they have enabled end user auth for the team (which located in another
    // part of the UI) and then try to delete the idp settings here?)
    return formErrors;
  }
  Object.entries(ERROR_CONFIGS).forEach(([k, v]) => {
    if (!v.isValid(data)) {
      if (!formErrors) {
        formErrors = { [k as FormError]: v.message };
      } else {
        formErrors[k as FormError] = v.message;
      }
    }
  });
  return formErrors;
};

const IdpSection = () => {
  const { config } = useContext(AppContext);
  const { renderFlash } = useContext(NotificationContext);
  const [formData, setFormData] = useState<IIdpFormData>({
    idpName: config?.mdm.end_user_authentication?.idp_name || "",
    entityId: config?.mdm.end_user_authentication?.entity_id || "",
    metadataUrl: config?.mdm.end_user_authentication?.metadata_url || "",
    metadata: config?.mdm.end_user_authentication?.metadata || "",
  });
  const [formErrors, setFormErrors] = useState<FormErrors | null>(null);

  const onInputChange = useCallback(
    ({ name, value }: { name: FormName; value: string }) => {
      const newData = { ...formData, [name]: value };
      setFormData(newData);
      setFormErrors(validateForm(newData));
    },
    [formData]
  );

  const onSubmit = useCallback(
    async (e: React.FormEvent<SubmitEvent>) => {
      e.preventDefault();
      const newErrors = validateForm(formData);
      if (newErrors) {
        setFormErrors(newErrors);
        return;
      }

      try {
        await configAPI.update({
          mdm: {
            end_user_authentication: {
              idp_name: formData.idpName,
              entity_id: formData.entityId,
              metadata_url: formData.metadataUrl,
              metadata: formData.metadata,
            },
          },
        });
        renderFlash("success", "Successfully updated end user authentication!");
      } catch (err) {
        const ae = (typeof err === "object" ? err : {}) as AxiosResponse;
        if (ae.status === 422) {
          renderFlash(
            "error",
            `Couldn’t update: ${expandErrorReasonRequired(err)}.`
          );
          return;
        }
        renderFlash("error", "Couldn’t update. Please try again.");
      }
    },
    [formData, renderFlash]
  );

  return (
    <div className={baseClass}>
      <SectionHeader title="End user authentication" />
      <form>
        <p>
          Connect Fleet to your identity provider to require end users to
          authenticate when they first setup their new macOS hosts.{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience##end-user-authentication-and-eula"
            text="Learn more"
            newTab
          />
        </p>
        <InputField
          label="Identity provider name"
          onChange={onInputChange}
          name="idpName"
          value={formData.idpName}
          parseTarget
          error={formErrors?.idpName}
          tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
        />
        <InputField
          label="Entity ID"
          onChange={onInputChange}
          name="entityId"
          value={formData.entityId}
          parseTarget
          error={formErrors?.entityId}
          tooltip="The required entity ID is a URI that you use to identify Fleet when configuring the identity provider."
        />
        <InputField
          label="Metadata URL"
          helpText={
            <>
              If both <b>Metadata URL</b> and <b>Metadata</b> are specified,{" "}
              <b>Metadata URL</b> will be used.
            </>
          }
          onChange={onInputChange}
          name="metadataUrl"
          value={formData.metadataUrl}
          parseTarget
          error={formErrors?.metadataOrMetadataUrl || formErrors?.metadataUrl}
          tooltip="Metadata URL provided by the identity provider."
        />
        <InputField
          label="Metadata"
          type="textarea"
          onChange={onInputChange}
          name="metadata"
          value={formData.metadata}
          parseTarget
          error={formErrors?.metadataOrMetadataUrl}
          tooltip="Metadata XML provided by the identity provider."
        />
        <TooltipWrapper
          tipContent="Complete all required fields to save end user authentication."
          disableTooltip={!formErrors}
        >
          <Button
            disabled={!!formErrors}
            onClick={onSubmit}
            className="button-wrap"
          >
            Save
          </Button>
        </TooltipWrapper>
      </form>
    </div>
  );
};

export default IdpSection;

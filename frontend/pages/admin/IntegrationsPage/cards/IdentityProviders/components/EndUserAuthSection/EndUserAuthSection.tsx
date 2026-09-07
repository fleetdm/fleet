import React, { useContext, useEffect, useRef } from "react";
import { AxiosResponse } from "axios";
import { isEqual } from "lodash";

import { expandErrorReasonRequired } from "interfaces/errors";
import { IEndUserAuthentication } from "interfaces/config";
import configAPI from "services/entities/config";
import { AppContext } from "context/app";
import useFormValidation, { trimFormData } from "hooks/useFormValidation";

import InputField from "components/forms/fields/InputField";
import Button from "components/buttons/Button/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import CustomLink from "components/CustomLink";
import { notify } from "components/ToastNotification";

import {
  IFormDataIdp,
  isEmptyFormData,
  newFormDataIdp,
  validateEndUserAuthForm,
} from "./helpers";

const baseClass = "end-user-auth-section";

export interface IEndUserAuthSectionProps {
  /**
   * The saved config, from the page's own config query. AppContext holds a
   * separate copy that the refetch after a save doesn't update, so seeding
   * from there shows pre-save values once the card remounts.
   */
  endUserAuth?: IEndUserAuthentication;
  onDirtyChange: (hasUnsavedChanges: boolean) => void;
  onSubmit: () => void;
}

const EndUserAuthSection = ({
  endUserAuth,
  onDirtyChange,
  // Notify parent component of changes, since we're calling our own API
  // rather than using the common config update handler.
  onSubmit: announceChanges,
}: IEndUserAuthSectionProps) => {
  const { config, isPremiumTier } = useContext(AppContext);
  const gitOpsModeEnabled = config?.gitops.gitops_mode_enabled;

  const originalFormData = useRef(newFormDataIdp(endUserAuth));

  const {
    formData,
    setField,
    reset,
    getError,
    clearFieldError,
    validateField,
    handleSubmit,
    clearErrors,
    isSubmitting,
  } = useFormValidation<IFormDataIdp>({
    initialFormData: originalFormData.current,
    validate: validateEndUserAuthForm,
  });

  const hasUnsavedChanges = !isEqual(
    trimFormData(formData),
    originalFormData.current
  );

  const onFieldChange = (name: keyof IFormDataIdp, value: string) => {
    setField(name, value);
    // Emptying the last field is how the configuration gets cleared, and an
    // empty form is valid, so every required-field error stops applying.
    if (isEmptyFormData({ ...formData, [name]: value })) {
      clearErrors();
    }
  };

  useEffect(() => {
    onDirtyChange(hasUnsavedChanges);
  }, [hasUnsavedChanges, onDirtyChange]);

  const onValidSubmit = async (submitData: IFormDataIdp) => {
    try {
      await configAPI.update({
        mdm: {
          end_user_authentication: {
            ...submitData,
          },
        },
      });
      notify.success("Successfully updated end user authentication.");
      originalFormData.current = submitData;
      reset(submitData);
      announceChanges();
    } catch (err) {
      const ae = (typeof err === "object" ? err : {}) as AxiosResponse;
      if (ae.status === 422) {
        notify.error(`Couldn't update: ${expandErrorReasonRequired(err)}.`, {
          response: err,
        });
        return;
      }
      notify.error("Couldn't update. Please try again.", { response: err });
    }
  };

  const renderContent = () => {
    if (!isPremiumTier) {
      return <PremiumFeatureMessage />;
    }

    return (
      <form onSubmit={handleSubmit(onValidSubmit)}>
        <p>
          After configuring, head to{" "}
          <strong>
            Controls &gt; Setup experience &gt; End user authentication
          </strong>{" "}
          to require end users to authenticate.{" "}
          <CustomLink
            text="Learn more"
            url="https://fleetdm.com/learn-more-about/end-user-authentication"
            newTab
          />
        </p>
        <div
          className={`form ${
            gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
          }`}
        >
          <InputField
            label="Identity provider name"
            name="idp_name"
            value={formData.idp_name}
            error={getError("idp_name")}
            onChange={(value: string) => onFieldChange("idp_name", value)}
            onFocus={() => clearFieldError("idp_name")}
            onBlur={() => validateField("idp_name")}
            disabled={isSubmitting}
            tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
          />
          <InputField
            label="Entity ID"
            name="entity_id"
            value={formData.entity_id}
            error={getError("entity_id")}
            onChange={(value: string) => onFieldChange("entity_id", value)}
            onFocus={() => clearFieldError("entity_id")}
            onBlur={() => validateField("entity_id")}
            disabled={isSubmitting}
            tooltip="The Entity ID is a required URI that you use to identify Fleet when configuring the identity provider. Okta calls this Audience Restriction."
          />
          <InputField
            label="Metadata URL"
            helpText={
              <>
                If both <b>Metadata URL</b> and <b>Metadata</b> are specified,{" "}
                <b>Metadata URL</b> will be used.
              </>
            }
            name="metadata_url"
            value={formData.metadata_url}
            error={getError("metadata_url")}
            onChange={(value: string) => onFieldChange("metadata_url", value)}
            onFocus={() => clearFieldError("metadata_url")}
            onBlur={() => validateField("metadata_url")}
            disabled={isSubmitting}
            tooltip="Metadata URL provided by the identity provider."
          />
          <InputField
            label="Metadata"
            type="textarea"
            name="metadata"
            value={formData.metadata}
            error={getError("metadata")}
            onChange={(value: string) => onFieldChange("metadata", value)}
            onFocus={() => clearFieldError("metadata")}
            onBlur={() => validateField("metadata")}
            disabled={isSubmitting}
            tooltip="Metadata XML provided by the identity provider."
          />
        </div>
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={isSubmitting || disableChildren}
              isLoading={isSubmitting}
              className="button-wrap"
            >
              Save
            </Button>
          )}
        />
      </form>
    );
  };

  return <div className={baseClass}>{renderContent()}</div>;
};

export default EndUserAuthSection;

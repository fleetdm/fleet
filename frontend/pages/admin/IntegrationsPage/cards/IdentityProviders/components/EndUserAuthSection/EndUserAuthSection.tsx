import { AxiosResponse } from "axios";
import { isEqual } from "lodash";
import React, { useContext, useEffect, useRef } from "react";

import Button from "components/buttons/Button/Button";
import CustomLink from "components/CustomLink";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import useFormValidation, { trimFormData } from "hooks/useFormValidation";
import { IEndUserAuthentication } from "interfaces/config";
import { expandErrorReasonRequired } from "interfaces/errors";
import configAPI from "services/entities/config";

import {
  IFormDataIdp,
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
    reset,
    getFieldProps,
    handleSubmit,
    isSubmitting,
  } = useFormValidation<IFormDataIdp>({
    initialFormData: originalFormData.current,
    validate: validateEndUserAuthForm,
  });

  const hasUnsavedChanges = !isEqual(
    trimFormData(formData),
    originalFormData.current
  );

  useEffect(() => {
    onDirtyChange(hasUnsavedChanges);
  }, [hasUnsavedChanges, onDirtyChange]);

  useEffect(() => () => onDirtyChange(false), [onDirtyChange]);

  const onValidSubmit = async (submitData: IFormDataIdp) => {
    // The fields and the button are disabled in GitOps mode.
    // Prevent the form element itself from being submitted.
    if (gitOpsModeEnabled) {
      return;
    }

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
            {...getFieldProps("idp_name")}
            disabled={isSubmitting || gitOpsModeEnabled}
            tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
          />
          <InputField
            label="Entity ID"
            {...getFieldProps("entity_id")}
            disabled={isSubmitting || gitOpsModeEnabled}
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
            {...getFieldProps("metadata_url")}
            disabled={isSubmitting || gitOpsModeEnabled}
            tooltip="Metadata URL provided by the identity provider."
          />
          <InputField
            label="Metadata"
            type="textarea"
            {...getFieldProps("metadata")}
            disabled={isSubmitting || gitOpsModeEnabled}
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

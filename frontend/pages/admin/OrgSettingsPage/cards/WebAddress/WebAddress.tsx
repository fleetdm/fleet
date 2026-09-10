import React from "react";

import useFormValidation, { IFormErrors } from "hooks/useFormValidation";

import SettingsSection from "pages/admin/components/SettingsSection";
import Button from "components/buttons/Button";
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import INVALID_SERVER_URL_MESSAGE from "utilities/error_messages";

import { IAppConfigFormProps } from "../constants";

interface IWebAddressFormData {
  serverURL: string;
}

const validate = ({ serverURL }: IWebAddressFormData): IFormErrors => {
  const errors: IFormErrors = {};
  if (!serverURL) {
    errors.serverURL = "Enter your Fleet web address";
  } else if (
    !validUrl({
      url: serverURL,
      protocols: ["http", "https"],
      allowLocalHost: true,
    })
  ) {
    errors.serverURL = INVALID_SERVER_URL_MESSAGE;
  }
  return errors;
};

const baseClass = "app-config-form";

const WebAddress = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

  const {
    formData,
    setField,
    getError,
    clearFieldError,
    validateField,
    handleSubmit: onSubmit,
    isSubmitting,
  } = useFormValidation<IWebAddressFormData>({
    initialFormData: {
      serverURL: appConfig.server_settings.server_url || "",
    },
    validate,
    isSubmitting: isUpdatingSettings,
  });

  const onValidSubmit = (data: IWebAddressFormData) =>
    handleSubmit({
      server_settings: {
        server_url: data.serverURL,
      },
    });

  return (
    <SettingsSection className={baseClass} title="Fleet web address">
      <form onSubmit={onSubmit(onValidSubmit)} autoComplete="off">
        <InputField
          label="URL"
          helpText={
            <>
              Include base path only (eg. no <code>/latest</code>)
            </>
          }
          name="serverURL"
          value={formData.serverURL}
          onChange={(value: string) => setField("serverURL", value)}
          onFocus={() => clearFieldError("serverURL")}
          onBlur={() => validateField("serverURL")}
          error={getError("serverURL")}
          tooltip="The base URL of this instance for use in Fleet links."
          disabled={gitOpsModeEnabled || isSubmitting}
        />
        <GitOpsModeTooltipWrapper
          renderChildren={(disableChildren) => (
            <Button
              type="submit"
              disabled={disableChildren || isSubmitting}
              className="button-wrap"
              isLoading={isSubmitting}
            >
              Save
            </Button>
          )}
        />
      </form>
    </SettingsSection>
  );
};

export default WebAddress;

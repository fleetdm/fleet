import React, { useState } from "react";
import { size } from "lodash";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import INVALID_SERVER_URL_MESSAGE from "utilities/error_messages";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IWebAddressFormData {
  serverURL: string;
}

interface IWebAddressFormErrors {
  server_url?: string | null;
}
const baseClass = "app-config-form";

const validateFormData = ({ serverURL }: IWebAddressFormData) => {
  const errors: IWebAddressFormErrors = {};
  if (!serverURL) {
    errors.server_url = "Fleet server URL must be present";
  } else if (
    !validUrl({
      url: serverURL,
      protocols: ["http", "https"],
      allowLocalHost: true,
    })
  ) {
    errors.server_url = INVALID_SERVER_URL_MESSAGE;
  }

  return errors;
};

const WebAddress = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

  const [formData, setFormData] = useState<IWebAddressFormData>({
    serverURL: appConfig.server_settings.server_url || "",
  });

  const { serverURL } = formData;

  const [formErrors, setFormErrors] = useState<IWebAddressFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    const newFormData = { ...formData, [name]: value };
    setFormData(newFormData);
    const newErrs = validateFormData(newFormData);
    // only set errors that are updates of existing errors
    // new errors are only set onBlur
    const errsToSet: Record<string, string> = {};
    Object.keys(formErrors).forEach((k) => {
      // @ts-ignore
      if (newErrs[k]) {
        // @ts-ignore
        errsToSet[k] = newErrs[k];
      }
    });
    setFormErrors(errsToSet);
  };

  const onInputBlur = () => {
    setFormErrors(validateFormData(formData));
  };

  const onFormSubmit = (evt: React.FormEvent<HTMLFormElement>) => {
    evt.preventDefault();
    // return null if there are errors
    const errs = validateFormData(formData);
    if (size(errs)) {
      setFormErrors(errs);
      return;
    }

    // Formatting of API not UI
    const formDataToSubmit = {
      server_settings: {
        server_url: serverURL,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Fleet web address" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <InputField
            label="URL"
            helpText={
              <>
                Include base path only (eg. no <code>/latest</code>)
              </>
            }
            onChange={onInputChange}
            name="serverURL"
            value={serverURL}
            parseTarget
            onBlur={onInputBlur}
            error={formErrors.server_url}
            tooltip="The base URL of this instance for use in Fleet links."
            disabled={gitOpsModeEnabled}
          />
          <GitOpsModeTooltipWrapper
            tipOffset={-8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={!!size(formErrors) || disableChildren}
                className="button-wrap"
                isLoading={isUpdatingSettings}
              >
                Save
              </Button>
            )}
          />
        </form>
      </div>
    </div>
  );
};

export default WebAddress;

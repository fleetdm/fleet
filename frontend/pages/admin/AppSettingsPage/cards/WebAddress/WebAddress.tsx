import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

const baseClass = "app-config-form";

const WebAddress = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<any>({
    serverURL: appConfig.server_settings.server_url || "",
  });

  const { serverURL } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (!serverURL) {
      errors.server_url = "Fleet server URL must be present";
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      server_settings: {
        server_url: serverURL,
        live_query_disabled: appConfig.server_settings.live_query_disabled,
        enable_analytics: appConfig.server_settings.enable_analytics,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>Fleet web address</h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Fleet app URL"
            hint={
              <span>
                Include base path only (eg. no <code>/latest</code>)
              </span>
            }
            onChange={handleInputChange}
            name="serverURL"
            value={serverURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server_url}
            tooltip="The base URL of this instance for use in Fleet links."
          />
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
        className="save-loading"
        spinner={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default WebAddress;

import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IWebAddressFormData {
  serverURL: string;
}

interface IWebAddressFormErrors {
  server_url?: string | null;
}

const baseClass = "app-config-form";

const WebAddress = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IWebAddressFormData>({
    serverURL: appConfig.server_settings.server_url || "",
  });

  const { serverURL } = formData;

  const [formErrors, setFormErrors] = useState<IWebAddressFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IWebAddressFormErrors = {};
    if (!serverURL) {
      errors.server_url = "Fleet server URL must be present";
    } else if (!validUrl({ url: serverURL, protocols: ["http", "https"] })) {
      errors.server_url = `${serverURL} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

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
            label="Fleet app URL"
            helpText={
              <>
                Include base path only (eg. no <code>/latest</code>)
              </>
            }
            onChange={onInputChange}
            name="serverURL"
            value={serverURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.server_url}
            tooltip="The base URL of this instance for use in Fleet links."
          />
          <Button
            type="submit"
            variant="brand"
            disabled={Object.keys(formErrors).length > 0}
            className="button-wrap"
            isLoading={isUpdatingSettings}
          >
            Save
          </Button>
        </form>
      </div>
    </div>
  );
};

export default WebAddress;

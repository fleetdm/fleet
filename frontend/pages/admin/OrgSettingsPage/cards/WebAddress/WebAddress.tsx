import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import SectionHeader from "components/SectionHeader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IWebAddressFormData {
  serverURL: string;
}

const baseClass = "app-config-form";

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

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
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
            tooltip="The base URL of this instance for use in Fleet links."
            disabled={gitOpsModeEnabled}
          />
          <GitOpsModeTooltipWrapper
            tipOffset={-8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                variant="brand"
                disabled={disableChildren}
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

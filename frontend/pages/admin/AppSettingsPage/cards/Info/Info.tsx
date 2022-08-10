import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import validUrl from "components/forms/validators/valid_url";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

const baseClass = "app-config-form";

const Info = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<any>({
    orgName: appConfig.org_info.org_name || "",
    orgLogoURL: appConfig.org_info.org_logo_url || "",
  });

  const { orgName, orgLogoURL } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (orgLogoURL && !validUrl(orgLogoURL)) {
      errors.org_logo_url = `${orgLogoURL} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      org_info: {
        org_logo_url: orgLogoURL,
        org_name: orgName,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>Organization info</h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Organization name"
            onChange={handleInputChange}
            name="orgName"
            value={orgName}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_name}
          />
          <InputField
            label="Organization avatar URL"
            onChange={handleInputChange}
            name="orgLogoURL"
            value={orgLogoURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_logo_url}
          />
        </div>
        <div className={`${baseClass}__details ${baseClass}__avatar-preview`}>
          <OrgLogoIcon src={orgLogoURL} />
        </div>
      </div>
      <Button
        type="submit"
        variant="brand"
        disabled={Object.keys(formErrors).length > 0}
        className="save-loading"
        isLoading={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default Info;

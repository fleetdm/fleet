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

interface IOrgInfoFormData {
  orgName: string;
  orgLogoURL: string;
  orgSupportURL: string;
}

const baseClass = "app-config-form";

const Info = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IOrgInfoFormData>({
    orgName: appConfig.org_info.org_name || "",
    orgLogoURL: appConfig.org_info.org_logo_url || "",
    orgSupportURL:
      appConfig.org_info.contact_url || "https://fleetdm.com/company/contact",
  });

  const { orgName, orgLogoURL, orgSupportURL } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (orgLogoURL && !validUrl({ url: orgLogoURL, protocol: "http" })) {
      errors.org_logo_url = `${orgLogoURL} is not a valid URL`;
    }

    if (!orgSupportURL) {
      errors.org_support_url = `Organization support URL must be present`;
    } else if (!validUrl({ url: orgSupportURL, protocol: "http" })) {
      errors.org_support_url = `${orgSupportURL} is not a valid URL`;
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
        contact_url: orgSupportURL,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section org-info`}>
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
          <InputField
            label="Organization support URL"
            onChange={handleInputChange}
            name="orgSupportURL"
            value={orgSupportURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_support_url}
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

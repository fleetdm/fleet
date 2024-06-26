import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import { IAppConfigFormProps, IFormField } from "../constants";

interface IOrgInfoFormData {
  orgLogoURL: string;
  orgName: string;
  orgLogoURLLightBackground: string;
  orgSupportURL: string;
}

interface IOrgInfoFormErrors {
  org_name?: string | null;
  org_logo_url?: string | null;
  org_logo_url_light_background?: string | null;
  org_support_url?: string | null;
}

// TODO: change base classes to these cards to follow the same pattern as the
// other components in the app.
const baseClass = "app-config-form";
const cardClass = "org-info";

const Info = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<IOrgInfoFormData>({
    orgName: appConfig.org_info.org_name || "",
    orgLogoURL: appConfig.org_info.org_logo_url || "",
    orgLogoURLLightBackground:
      appConfig.org_info.org_logo_url_light_background || "",
    orgSupportURL:
      appConfig.org_info.contact_url || "https://fleetdm.com/company/contact",
  });

  const {
    orgName,
    orgLogoURL,
    orgLogoURLLightBackground,
    orgSupportURL,
  } = formData;

  const [formErrors, setFormErrors] = useState<IOrgInfoFormErrors>({});

  const onInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IOrgInfoFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (
      orgLogoURL &&
      !validUrl({ url: orgLogoURL, protocols: ["http", "https"] })
    ) {
      errors.org_logo_url = `${orgLogoURL} is not a valid URL`;
    }

    if (!orgSupportURL) {
      errors.org_support_url = `Organization support URL must be present`;
    } else if (
      !validUrl({ url: orgSupportURL, protocols: ["http", "https"] })
    ) {
      errors.org_support_url = `${orgSupportURL} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataToSubmit = {
      org_info: {
        org_logo_url: orgLogoURL,
        org_logo_url_light_background: orgLogoURLLightBackground,
        org_name: orgName,
        contact_url: orgSupportURL,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section ${cardClass}`}>
        <SectionHeader title="Organization info" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <InputField
            label="Organization name"
            onChange={onInputChange}
            name="orgName"
            value={orgName}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_name}
          />
          <InputField
            label="Organization support URL"
            onChange={onInputChange}
            name="orgSupportURL"
            value={orgSupportURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.org_support_url}
          />
          <div className={`${cardClass}__logo-field-set`}>
            <InputField
              label="Organization avatar URL (for dark backgrounds)"
              onChange={onInputChange}
              name="orgLogoURL"
              value={orgLogoURL}
              parseTarget
              onBlur={validateForm}
              error={formErrors.org_logo_url}
              inputWrapperClass={`${cardClass}__logo-field`}
              tooltip="Logo is displayed in the top bar and other areas of Fleet that
                have dark backgrounds."
            />
            <div
              className={`${cardClass}__icon-preview ${cardClass}__dark-background`}
            >
              <OrgLogoIcon
                className={`${cardClass}__icon-img`}
                src={orgLogoURL}
              />
            </div>
          </div>
          <div className={`${cardClass}__logo-field-set`}>
            <InputField
              label="Organization avatar URL (for light backgrounds)"
              onChange={onInputChange}
              name="orgLogoURLLightBackground"
              value={orgLogoURLLightBackground}
              parseTarget
              onBlur={validateForm}
              error={formErrors.org_logo_url_light_background}
              inputWrapperClass={`${cardClass}__logo-field`}
              tooltip="Logo is displayed in Fleet on top of light backgrounds.
"
            />
            <div
              className={`${cardClass}__icon-preview ${cardClass}__light-background`}
            >
              <OrgLogoIcon
                className={`${cardClass}__icon-img`}
                src={orgLogoURLLightBackground}
              />
            </div>
          </div>
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

export default Info;

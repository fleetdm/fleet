import React, { useState } from "react";

import isDataURI from "validator/lib/isDataURI";

import Button from "components/buttons/Button";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
// @ts-ignore
import OrgLogoIcon from "components/icons/OrgLogoIcon";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import TooltipWrapper from "components/TooltipWrapper";

import { IParseTargetFormField } from "components/forms/fields/InputField/helpers";
import { IAppConfigFormProps } from "../constants";

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

const validateOrgLogoURL = (url: string) =>
  isDataURI(url) || validUrl({ url, protocols: ["http", "https"] });

const Info = ({
  appConfig,
  handleSubmit,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

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

  const onInputChange = ({ name, value }: IParseTargetFormField) => {
    setFormData({ ...formData, [name]: value });
    setFormErrors({});
  };

  const validateForm = () => {
    const errors: IOrgInfoFormErrors = {};

    if (!orgName) {
      errors.org_name = "Organization name must be present";
    }

    if (orgLogoURL && !validateOrgLogoURL(orgLogoURL)) {
      errors.org_logo_url = `${orgLogoURL} is not a valid URL`;
    }

    if (
      orgLogoURLLightBackground &&
      !validateOrgLogoURL(orgLogoURLLightBackground)
    ) {
      errors.org_logo_url_light_background = `${orgLogoURL} is not a valid URL`;
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
          {/* "form" class applies global form styling to fields for free */}
          <div
            className={`form ${
              gitOpsModeEnabled ? "disabled-by-gitops-mode" : ""
            }`}
          >
            <p className={`${baseClass}__section-description`}>
              This logo is displayed in the top navigation, setup experience
              window, and MDM migration dialog. Please use{" "}
              <CustomLink
                url="https://fleetdm.com/learn-more-about/organization-logo-size"
                text="recommended sizes"
                newTab
                multiline
              />
            </p>
            <div className={`${cardClass}__logo-field-set`}>
              <InputField
                label="Logo URL for dark background"
                onChange={onInputChange}
                name="orgLogoURL"
                value={orgLogoURL}
                parseTarget
                onBlur={validateForm}
                error={formErrors.org_logo_url}
                inputWrapperClass={`${cardClass}__logo-field`}
                tooltip={
                  <>
                    Logo is displayed in the top bar and other
                    <br />
                    areas of Fleet that have dark backgrounds.
                  </>
                }
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
                label="Logo URL for light background"
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
              label={
                <TooltipWrapper
                  tipContent={
                    <>
                      URL is used in &quot;Reach out to IT&quot; links shown to
                      the end
                      <br />
                      user (e.g. self-service and during MDM migration).
                    </>
                  }
                >
                  Organization support URL
                </TooltipWrapper>
              }
              onChange={onInputChange}
              name="orgSupportURL"
              value={orgSupportURL}
              parseTarget
              onBlur={validateForm}
              error={formErrors.org_support_url}
            />
          </div>
          <GitOpsModeTooltipWrapper
            tipOffset={-8}
            renderChildren={(disableChildren) => (
              <Button
                type="submit"
                disabled={Object.keys(formErrors).length > 0 || disableChildren}
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

export default Info;

import React, { useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import validHostname from "components/forms/validators/valid_hostname";

import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import CustomLink from "components/CustomLink";

import { DEFAULT_TRANSPARENCY_URL, IAppConfigFormProps } from "../constants";
import TooltipWrapper from "../../../../../components/TooltipWrapper";

interface IFleetDesktopFormData {
  transparencyURL: string;
  alternativeBrowserHost: string;
}
interface IFleetDesktopFormErrors {
  transparencyURL?: string | null;
  alternativeBrowserHost?: string | null;
}
const baseClass = "app-config-form";

const FleetDesktop = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const gitOpsModeEnabled = appConfig.gitops.gitops_mode_enabled;

  const [formData, setFormData] = useState<IFleetDesktopFormData>({
    transparencyURL:
      appConfig.fleet_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
    alternativeBrowserHost:
      appConfig.fleet_desktop?.alternative_browser_host || "",
  });

  const [formErrors, setFormErrors] = useState<IFleetDesktopFormErrors>({});

  const onInputChange = ({ name, value }: IInputFieldParseTarget) => {
    setFormData((prevFormData) => ({
      ...prevFormData,
      [name]: value.toString(),
    }));
    setFormErrors((prevErrors) => {
      const newErrors = { ...prevErrors };
      delete newErrors[name as keyof IFleetDesktopFormErrors];
      return newErrors;
    });
  };

  const validateForm = () => {
    const { transparencyURL, alternativeBrowserHost } = formData;

    const errors: IFleetDesktopFormErrors = {};

    if (
      transparencyURL &&
      !validUrl({ url: transparencyURL, protocols: ["http", "https"] })
    ) {
      errors.transparencyURL = `Custom transparency URL must include protocol (e.g. https://)`;
    }

    if (alternativeBrowserHost && !validHostname(alternativeBrowserHost)) {
      errors.alternativeBrowserHost = `Browser host must be a valid hostname or IP address (e.g. example.com, 192.168.1.50) and may include a port.`;
    }

    setFormErrors(errors);
  };

  const getAlternativeBrowserHostUrlTooltip = () => (
    <>
      If you are using mTLS for your agent-server communication, specify an
      alternative host to direct Fleet Desktop through.
      <CustomLink
        url="https://fleetdm.com/learn-more-about/alternative-browser-host"
        text="Learn more "
        variant="tooltip-link"
        newTab
      />
    </>
  );

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataForAPI = {
      fleet_desktop: {
        transparency_url: formData.transparencyURL,
        alternative_browser_host: formData.alternativeBrowserHost,
      },
    };

    handleSubmit(formDataForAPI);
  };

  if (!isPremiumTier) {
    return <></>;
  }

  return (
    <SettingsSection title="Fleet Desktop">
      <PageDescription
        variant="right-panel"
        content={
          <>Override default URLs to customize the Fleet Desktop experience.</>
        }
      />
      <form onSubmit={onFormSubmit} autoComplete="off">
        <InputField
          label="Custom transparency URL"
          onChange={onInputChange}
          name="transparencyURL"
          value={formData.transparencyURL}
          parseTarget
          onBlur={validateForm}
          error={formErrors.transparencyURL}
          placeholder="https://fleetdm.com/transparency"
          disabled={gitOpsModeEnabled}
          helpText={
            <>
              {" "}
              By default, end users who click &quot;About Fleet&quot; in the
              Fleet Desktop menu are taken to{" "}
              <CustomLink
                url="https://fleetdm.com/transparency"
                text="https://fleetdm.com/transparency"
                newTab
                multiline
              />{" "}
            </>
          }
        />
        <InputField
          label={
            <TooltipWrapper tipContent={getAlternativeBrowserHostUrlTooltip()}>
              Browser host
            </TooltipWrapper>
          }
          onChange={onInputChange}
          onBlur={validateForm}
          name="alternativeBrowserHost"
          value={formData.alternativeBrowserHost}
          parseTarget
          error={formErrors.alternativeBrowserHost}
          disabled={gitOpsModeEnabled}
          helpText="If not set, Fleet Desktop uses your Fleet web address."
        />
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
    </SettingsSection>
  );
};

export default FleetDesktop;

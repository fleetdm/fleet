import React, { useState } from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import CustomLink from "components/CustomLink";

import { DEFAULT_TRANSPARENCY_URL, IAppConfigFormProps } from "../constants";

interface IFleetDesktopFormData {
  transparencyUrl: string;
}
interface IFleetDesktopFormErrors {
  transparency_url?: string | null;
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
    transparencyUrl:
      appConfig.fleet_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
  });

  const [formErrors, setFormErrors] = useState<IFleetDesktopFormErrors>({});

  const onInputChange = ({ value }: IInputFieldParseTarget) => {
    setFormData({ transparencyUrl: value.toString() });
    setFormErrors({});
  };

  const validateForm = () => {
    const { transparencyUrl } = formData;

    const errors: IFleetDesktopFormErrors = {};
    if (
      transparencyUrl &&
      !validUrl({ url: transparencyUrl, protocols: ["http", "https"] })
    ) {
      errors.transparency_url = `Custom transparency URL must include protocol (e.g. https://)`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataForAPI = {
      fleet_desktop: {
        transparency_url: formData.transparencyUrl,
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
          name="transparency_url"
          value={formData.transparencyUrl}
          parseTarget
          onBlur={validateForm}
          error={formErrors.transparency_url}
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

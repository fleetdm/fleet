import React, { useState } from "react";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import CustomLink from "components/CustomLink";
import {
  DEFAULT_TRANSPARENCY_URL,
  IAppConfigFormProps,
  IFormField,
} from "../constants";

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
  const [formData, setFormData] = useState<IFleetDesktopFormData>({
    transparencyUrl:
      appConfig.fleet_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
  });

  const [formErrors, setFormErrors] = useState<IFleetDesktopFormErrors>({});

  const onInputChange = ({ value }: IFormField) => {
    setFormData({ transparencyUrl: value.toString() });
    setFormErrors({});
  };

  const validateForm = () => {
    const { transparencyUrl } = formData;

    const errors: IFleetDesktopFormErrors = {};
    if (transparencyUrl && !validUrl({ url: transparencyUrl })) {
      errors.transparency_url = `${transparencyUrl} is not a valid URL`;
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
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Fleet Desktop" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <p className={`${baseClass}__section-description`}>
            When an end user clicks “About Fleet” in the Fleet Desktop menu, by
            default they are taken to{" "}
            <CustomLink
              url="https://fleetdm.com/transparency"
              text="https://fleetdm.com/transparency"
              newTab
              multiline
            />{" "}
            . You can override the URL to take them to a resource of your
            choice.
          </p>
          <InputField
            label="Custom transparency URL"
            onChange={onInputChange}
            name="transparency_url"
            value={formData.transparencyUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.transparency_url}
            placeholder="https://fleetdm.com/transparency"
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

export default FleetDesktop;

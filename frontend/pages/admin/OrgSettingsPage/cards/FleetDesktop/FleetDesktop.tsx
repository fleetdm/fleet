import React, { useState } from "react";

import { IConfig, IConfigFormData } from "interfaces/config";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";

import CustomLink from "components/CustomLink";
import {
  DEFAULT_TRANSPARENCY_URL,
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

const baseClass = "app-config-form";

const FleetDesktop = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<
    Pick<IConfigFormData, "transparencyUrl">
  >({
    transparencyUrl:
      appConfig.fleet_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
  });

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ value }: IFormField) => {
    setFormData({ transparencyUrl: value.toString() });
    setFormErrors({});
  };

  const validateForm = () => {
    const { transparencyUrl } = formData;

    const errors: IAppConfigFormErrors = {};
    if (transparencyUrl && !validUrl({ url: transparencyUrl })) {
      errors.transparency_url = `${transparencyUrl} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataForAPI: Pick<IConfig, "fleet_desktop"> = {
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
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>Fleet Desktop</h2>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Custom transparency URL"
            onChange={handleInputChange}
            name="transparency_url"
            value={formData.transparencyUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.transparency_url}
            placeholder="https://fleetdm.com/transparency"
          />
          <p className={`${baseClass}__component-details`}>
            When an end user clicks “Transparency” in the Fleet Desktop menu, by
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

export default FleetDesktop;

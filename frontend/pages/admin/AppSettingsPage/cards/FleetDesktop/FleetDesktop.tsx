import React, { useState } from "react";

import { IConfig, IConfigFormData } from "interfaces/config";

import Button from "components/buttons/Button";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";

import OpenNewTabIcon from "../../../../../../assets/images/open-new-tab-12x12@2x.png";

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
    Pick<IConfigFormData, "transparency_url">
  >({
    transparency_url:
      appConfig.fleet_desktop?.transparency_url || DEFAULT_TRANSPARENCY_URL,
  });

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ value }: IFormField) => {
    setFormData({ transparency_url: value.toString() });
    setFormErrors({});
  };

  const validateForm = () => {
    const { transparency_url } = formData;

    const errors: IAppConfigFormErrors = {};
    if (!transparency_url) {
      errors.transparency_url = "Transparency URL name must be present";
    } else if (!validUrl(transparency_url)) {
      errors.transparency_url = `${transparency_url} is not a valid URL`;
    }

    setFormErrors(errors);
  };

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    const formDataForAPI: Pick<IConfig, "fleet_desktop"> = {
      fleet_desktop: {
        transparency_url: formData.transparency_url,
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
            value={formData.transparency_url}
            parseTarget
            onBlur={validateForm}
            error={formErrors.transparency_url}
          />
          <p className={`${baseClass}__component-label`}>
            When an end user clicks “Transparency” in the Fleet Desktop menu, by
            default they are taken to{" "}
            <a
              className={`${baseClass}__transparency`}
              href="https://fleetdm.com/transparency"
              target="_blank"
              rel="noopener noreferrer"
            >
              {" "}
              https://fleetdm.com/transparency{" "}
              <img className="icon" src={OpenNewTabIcon} alt="open new tab" />
            </a>{" "}
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
        spinner={isUpdatingSettings}
      >
        Save
      </Button>
    </form>
  );
};

export default FleetDesktop;

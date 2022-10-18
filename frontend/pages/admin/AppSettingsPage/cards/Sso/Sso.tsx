import React, { useState, useEffect } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

import ExternalLinkIcon from "../../../../../../assets/images/icon-external-link-12x12@2x.png";

const baseClass = "app-config-form";

interface ISsoFormData {
  enableSSO?: boolean;
  idpName?: string;
  entityID?: string;
  issuerURI?: string;
  idpImageURL?: string;
  metadata?: string;
  metadataURL?: string;
  enableSSOIDPLogin?: boolean;
  enableJITProvisioning?: boolean;
}

const Sso = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<ISsoFormData>({
    enableSSO: appConfig.sso_settings.enable_sso ?? false,
    idpName: appConfig.sso_settings.idp_name ?? "",
    entityID: appConfig.sso_settings.entity_id ?? "",
    issuerURI: appConfig.sso_settings.issuer_uri ?? "",
    idpImageURL: appConfig.sso_settings.idp_image_url ?? "",
    metadata: appConfig.sso_settings.metadata ?? "",
    metadataURL: appConfig.sso_settings.metadata_url ?? "",
    enableSSOIDPLogin: appConfig.sso_settings.enable_sso_idp_login ?? false,
    enableJITProvisioning:
      appConfig.sso_settings.enable_jit_provisioning ?? false,
  });

  const {
    enableSSO,
    idpName,
    entityID,
    issuerURI,
    idpImageURL,
    metadata,
    metadataURL,
    enableSSOIDPLogin,
    enableJITProvisioning,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableSSO) {
      if (idpImageURL && !validUrl(idpImageURL)) {
        errors.idp_image_url = `${idpImageURL} is not a valid URL`;
      }

      if (metadata === "" && metadataURL === "") {
        errors.metadata_url = "Metadata URL must be present";
      }

      if (!entityID) {
        errors.entity_id = "Entity ID must be present";
      }

      if (typeof entityID === "string" && entityID.length < 5) {
        errors.entity_id = "Entity ID must be 5 or more characters";
      }

      if (!idpName) {
        errors.idp_name = "Identity provider name must be present";
      }
    }

    setFormErrors(errors);
  };

  useEffect(() => {
    validateForm();
  }, [enableSSO]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      sso_settings: {
        entity_id: entityID?.trim(),
        issuer_uri: issuerURI?.trim(),
        idp_image_url: idpImageURL?.trim(),
        metadata: metadata?.trim(),
        metadata_url: metadataURL?.trim(),
        idp_name: idpName?.trim(),
        enable_sso: enableSSO,
        enable_sso_idp_login: enableSSOIDPLogin,
        enable_jit_provisioning: enableJITProvisioning,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <form className={baseClass} onSubmit={onFormSubmit} autoComplete="off">
      <div className={`${baseClass}__section`}>
        <h2>Single sign-on options</h2>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableSSO"
            value={enableSSO}
            parseTarget
          >
            Enable single sign-on
          </Checkbox>
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Identity provider name"
            onChange={handleInputChange}
            name="idpName"
            value={idpName}
            parseTarget
            onBlur={validateForm}
            error={formErrors.idp_name}
            tooltip="A required human friendly name for the identity provider that will provide single sign-on authentication."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Entity ID"
            hint={
              <span>
                The URI you provide here must exactly match the Entity ID field
                used in identity provider configuration.
              </span>
            }
            onChange={handleInputChange}
            name="entityID"
            value={entityID}
            parseTarget
            onBlur={validateForm}
            error={formErrors.entity_id}
            tooltip="The required entity ID is a URI that you use to identify Fleet when configuring the identity provider."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Issuer URI"
            onChange={handleInputChange}
            name="issuerURI"
            value={issuerURI}
            parseTarget
            tooltip="The issuer URI supplied by the identity provider."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="IDP image URL"
            onChange={handleInputChange}
            name="idpImageURL"
            value={idpImageURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.idp_image_url}
            tooltip="An optional link to an image such as a logo for the identity provider."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Metadata"
            type="textarea"
            onChange={handleInputChange}
            name="metadata"
            value={metadata}
            parseTarget
            onBlur={validateForm}
            tooltip="Metadata provided by the identity provider. Either metadata or a metadata url must be provided."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <InputField
            label="Metadata URL"
            hint={
              <span>
                If available from the identity provider, this is the preferred
                means of providing metadata.
              </span>
            }
            onChange={handleInputChange}
            name="metadataURL"
            value={metadataURL}
            parseTarget
            onBlur={validateForm}
            error={formErrors.metadata_url}
            tooltip="A URL that references the identity provider metadata."
          />
        </div>
        <div className={`${baseClass}__inputs`}>
          <Checkbox
            onChange={handleInputChange}
            name="enableSSOIDPLogin"
            value={enableSSOIDPLogin}
            parseTarget
          >
            Allow SSO login initiated by identity provider
          </Checkbox>
        </div>
        {isPremiumTier && (
          <div className={`${baseClass}__inputs`}>
            <Checkbox
              onChange={handleInputChange}
              name="enableJITProvisioning"
              value={enableJITProvisioning}
              parseTarget
            >
              <>
                Automatically create Observer user on login{" "}
                <a
                  href="https://fleetdm.com/docs/deploying/configuration?utm_medium=fleetui&utm_source=sso-settings#just-in-time-jit-user-provisioning"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Learn more
                  <img alt="Open external link" src={ExternalLinkIcon} />
                </a>
              </>
            </Checkbox>
          </div>
        )}
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

export default Sso;

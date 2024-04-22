import React, { useState, useEffect } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import CustomLink from "components/CustomLink";
// @ts-ignore
import InputField from "components/forms/fields/InputField";
import validUrl from "components/forms/validators/valid_url";
import SectionHeader from "components/SectionHeader";

import {
  IAppConfigFormProps,
  IFormField,
  IAppConfigFormErrors,
} from "../constants";

const baseClass = "app-config-form";

interface ISsoFormData {
  enableSso?: boolean;
  idpName?: string;
  entityId?: string;
  idpImageUrl?: string;
  metadata?: string;
  metadataUrl?: string;
  enableSsoIdpLogin?: boolean;
  enableJitProvisioning?: boolean;
}

const Sso = ({
  appConfig,
  handleSubmit,
  isPremiumTier,
  isUpdatingSettings,
}: IAppConfigFormProps): JSX.Element => {
  const [formData, setFormData] = useState<ISsoFormData>({
    enableSso: appConfig.sso_settings.enable_sso ?? false,
    idpName: appConfig.sso_settings.idp_name ?? "",
    entityId: appConfig.sso_settings.entity_id ?? "",
    idpImageUrl: appConfig.sso_settings.idp_image_url ?? "",
    metadata: appConfig.sso_settings.metadata ?? "",
    metadataUrl: appConfig.sso_settings.metadata_url ?? "",
    enableSsoIdpLogin: appConfig.sso_settings.enable_sso_idp_login ?? false,
    enableJitProvisioning:
      appConfig.sso_settings.enable_jit_provisioning ?? false,
  });

  const {
    enableSso,
    idpName,
    entityId,
    idpImageUrl,
    metadata,
    metadataUrl,
    enableSsoIdpLogin,
    enableJitProvisioning,
  } = formData;

  const [formErrors, setFormErrors] = useState<IAppConfigFormErrors>({});

  const handleInputChange = ({ name, value }: IFormField) => {
    setFormData({ ...formData, [name]: value });
  };

  const validateForm = () => {
    const errors: IAppConfigFormErrors = {};

    if (enableSso) {
      if (idpImageUrl && !validUrl({ url: idpImageUrl })) {
        errors.idp_image_url = `${idpImageUrl} is not a valid URL`;
      }

      if (!metadata) {
        if (!metadataUrl) {
          errors.metadata_url = "Metadata or Metadata URL must be present";
          errors.metadata = "Metadata or Metadata URL must be present";
        } else if (
          !validUrl({ url: metadataUrl, protocols: ["http", "https"] })
        ) {
          errors.metadata_url = `${metadataUrl} is not a valid URL`;
        }
      }

      if (!entityId) {
        errors.entity_id = "Entity ID must be present";
      }

      if (typeof entityId === "string" && entityId.length < 5) {
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
  }, [idpImageUrl, metadata, metadataUrl, entityId, idpName]);

  const onFormSubmit = (evt: React.MouseEvent<HTMLFormElement>) => {
    evt.preventDefault();

    // Formatting of API not UI
    const formDataToSubmit = {
      sso_settings: {
        entity_id: entityId?.trim(),
        idp_image_url: idpImageUrl?.trim(),
        metadata: metadata?.trim(),
        metadata_url: metadataUrl?.trim(),
        idp_name: idpName?.trim(),
        enable_sso: enableSso,
        enable_sso_idp_login: enableSsoIdpLogin,
        enable_jit_provisioning: enableJitProvisioning,
      },
    };

    handleSubmit(formDataToSubmit);
  };

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__section`}>
        <SectionHeader title="Single sign-on options" />
        <form onSubmit={onFormSubmit} autoComplete="off">
          <Checkbox
            onChange={handleInputChange}
            name="enableSso"
            value={enableSso}
            parseTarget
          >
            Enable single sign-on
          </Checkbox>
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
          <InputField
            label="Entity ID"
            helpText="The URI you provide here must exactly match the Entity ID field used in identity provider configuration."
            onChange={handleInputChange}
            name="entityId"
            value={entityId}
            parseTarget
            onBlur={validateForm}
            error={formErrors.entity_id}
            tooltip="The required entity ID is a URI that you use to identify Fleet when configuring the identity provider."
          />
          <InputField
            label="IDP image URL"
            onChange={handleInputChange}
            name="idpImageUrl"
            value={idpImageUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.idp_image_url}
            tooltip={`An optional link to an image such
            as a logo for the identity provider.`}
          />
          <InputField
            label="Metadata"
            type="textarea"
            onChange={handleInputChange}
            name="metadata"
            value={metadata}
            parseTarget
            onBlur={validateForm}
            error={formErrors.metadata}
            tooltip={`Metadata provided by the identity provider. Either
            metadata or a metadata url must be provided.`}
          />
          <InputField
            label="Metadata URL"
            helpText="If available from the identity provider, this is the preferred means of providing metadata."
            onChange={handleInputChange}
            name="metadataUrl"
            value={metadataUrl}
            parseTarget
            onBlur={validateForm}
            error={formErrors.metadata_url}
            tooltip="A URL that references the identity provider metadata."
          />
          <Checkbox
            onChange={handleInputChange}
            name="enableSsoIdpLogin"
            value={enableSsoIdpLogin}
            parseTarget
          >
            Allow SSO login initiated by identity provider
          </Checkbox>
          {isPremiumTier && (
            <Checkbox
              onChange={handleInputChange}
              name="enableJitProvisioning"
              value={enableJitProvisioning}
              parseTarget
            >
              <>
                Create user and sync permissions on login{" "}
                <CustomLink
                  url="https://fleetdm.com/learn-more-about/just-in-time-provisioning"
                  text="Learn more"
                  newTab
                />
              </>
            </Checkbox>
          )}
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

export default Sso;

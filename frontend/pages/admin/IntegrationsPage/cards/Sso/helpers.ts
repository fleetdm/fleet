import { IConfig } from "interfaces/config";
import { IFormErrors } from "hooks/useFormValidation";
import validUrl from "components/forms/validators/valid_url";

export interface ISsoFormData {
  idpName: string;
  enableSso: boolean;
  entityId: string;
  idpImageUrl: string;
  metadata: string;
  metadataUrl: string;
  enableSsoIdpLogin: boolean;
  enableJitProvisioning: boolean;
}

export const newSsoFormData = (appConfig: IConfig): ISsoFormData => ({
  enableSso: appConfig.sso_settings?.enable_sso ?? false,
  idpName: appConfig.sso_settings?.idp_name ?? "",
  entityId: appConfig.sso_settings?.entity_id ?? "",
  idpImageUrl: appConfig.sso_settings?.idp_image_url ?? "",
  metadata: appConfig.sso_settings?.metadata ?? "",
  metadataUrl: appConfig.sso_settings?.metadata_url ?? "",
  enableSsoIdpLogin: appConfig.sso_settings?.enable_sso_idp_login ?? false,
  enableJitProvisioning:
    appConfig.sso_settings?.enable_jit_provisioning ?? false,
});

export const validateSsoForm = (formData: ISsoFormData): IFormErrors => {
  const errors: IFormErrors = {};

  const {
    enableSso,
    idpImageUrl,
    metadata,
    metadataUrl,
    entityId,
    idpName,
  } = formData;

  if (!enableSso) {
    return errors;
  }

  if (idpImageUrl && !validUrl({ url: idpImageUrl })) {
    errors.idpImageUrl = "Enter a valid IdP image URL";
  }

  if (!metadata) {
    if (!metadataUrl) {
      errors.metadataUrl = "Enter metadata or a metadata URL";
      errors.metadata = "Enter metadata or a metadata URL";
    } else if (!validUrl({ url: metadataUrl, protocols: ["http", "https"] })) {
      errors.metadataUrl = "Enter a valid metadata URL";
    }
  }

  if (!entityId) {
    errors.entityId = "Enter an entity ID";
  }

  if (!idpName) {
    errors.idpName = "Enter an identity provider name";
  }

  return errors;
};

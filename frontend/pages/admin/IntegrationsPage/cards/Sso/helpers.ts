import { IConfig } from "interfaces/config";
import { IFormErrors, trimFormData } from "hooks/useFormValidation";
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
  idpName: appConfig.sso_settings?.idp_name?.trim() ?? "",
  entityId: appConfig.sso_settings?.entity_id?.trim() ?? "",
  idpImageUrl: appConfig.sso_settings?.idp_image_url?.trim() ?? "",
  metadata: appConfig.sso_settings?.metadata?.trim() ?? "",
  metadataUrl: appConfig.sso_settings?.metadata_url?.trim() ?? "",
  enableSsoIdpLogin: appConfig.sso_settings?.enable_sso_idp_login ?? false,
  enableJitProvisioning:
    appConfig.sso_settings?.enable_jit_provisioning ?? false,
});

export type SsoTextField =
  | "idpName"
  | "entityId"
  | "idpImageUrl"
  | "metadata"
  | "metadataUrl";

/**
 * Either field satisfies the metadata requirement, so they share one error
 * message. Only `metadataUrl` can also hold a URL-format error.
 */
export const METADATA_SIBLING: Partial<Record<SsoTextField, SsoTextField>> = {
  metadata: "metadataUrl",
  metadataUrl: "metadata",
};

export const validateSsoForm = (formData: ISsoFormData): IFormErrors => {
  const errors: IFormErrors = {};

  // Blur validates the raw values, so whitespace-only content has to read as
  // empty here rather than only once the hook trims on submit.
  const {
    enableSso,
    idpImageUrl,
    metadata,
    metadataUrl,
    entityId,
    idpName,
  } = trimFormData(formData);

  // Everything below only reaches the API when SSO is on, so an incomplete
  // config the user is leaving turned off isn't an error.
  if (!enableSso) {
    return errors;
  }

  if (idpImageUrl && !validUrl({ url: idpImageUrl })) {
    errors.idpImageUrl = "Enter a valid IdP image URL";
  }

  if (!metadata && !metadataUrl) {
    errors.metadataUrl = "Enter metadata or a metadata URL";
    errors.metadata = "Enter metadata or a metadata URL";
  } else if (metadataUrl) {
    // Checked whenever it has a value: when both are set the URL is the one
    // that gets used, so an invalid one can't ride along on valid metadata.
    if (!validUrl({ url: metadataUrl })) {
      errors.metadataUrl = "Enter a valid metadata URL";
    } else if (!validUrl({ url: metadataUrl, protocols: ["http", "https"] })) {
      errors.metadataUrl =
        "Enter a metadata URL starting with https:// or http://";
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

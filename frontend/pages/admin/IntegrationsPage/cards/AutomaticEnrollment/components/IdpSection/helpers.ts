import { IEndUserAuthentication } from "interfaces/config";

import validateUrl from "components/forms/validators/valid_url";

export interface IFormDataIdp {
  idp_name: string;
  entity_id: string;
  metadata_url: string;
  metadata: string;
}

export const newFormDataIdp = (
  config?: IEndUserAuthentication
): IFormDataIdp => {
  return {
    idp_name: config?.idp_name || "",
    entity_id: config?.entity_id || "",
    metadata_url: config?.metadata_url || "",
    metadata: config?.metadata || "",
  };
};

const isEmptyFormData = (data: IFormDataIdp) => {
  const values = Object.values(data);
  return !values.length || values.every((v) => v === "");
};

const errorIdpName = (data: IFormDataIdp) => {
  if (!data.idp_name) {
    return "Identity provider name must be present.";
  }
  return "";
};

const errorEntityId = (data: IFormDataIdp) => {
  if (!data.entity_id) {
    return "Entity ID must be present.";
  }
  if (data.entity_id.length < 5) {
    return "Entity ID must be 5 or more characters.";
  }
  return "";
};

const errorMetadataUrl = (data: IFormDataIdp) => {
  if (!data.metadata && !data.metadata_url) {
    return "Metadata or Metadata URL must be present.";
  }
  if (data.metadata_url && !validateUrl({ url: data.metadata_url })) {
    return `${data.metadata_url} is not a valid URL.`;
  }
  return "";
};

const errorMetadata = (data: IFormDataIdp) => {
  if (!data.metadata && !data.metadata_url) {
    return "Metadata or Metadata URL must be present.";
  }
  return "";
};

const validators = {
  idp_name: errorIdpName,
  entity_id: errorEntityId,
  metadata_url: errorMetadataUrl,
  metadata: errorMetadata,
} as const;

export type IFormErrorsIdp = Partial<Record<keyof IFormDataIdp, string>>;

export const validateFormDataIdp = (
  data: IFormDataIdp
): IFormErrorsIdp | null => {
  let formErrors: IFormErrorsIdp | null = null;
  if (isEmptyFormData(data)) {
    // TODO: confirm whether we want to allow user to save an empty form or if should be treated
    // as a form error (what happens is they have enabled end user auth for the team (which located in another
    // part of the UI) and then try to delete the idp settings here?)
    return formErrors;
  }
  Object.entries(validators).forEach(([k, v]) => {
    const err = v(data);
    if (err) {
      if (!formErrors) {
        formErrors = { [k as keyof IFormDataIdp]: err };
      } else {
        formErrors[k as keyof IFormDataIdp] = err;
      }
    }
  });
  return formErrors;
};

import { IEndUserAuthentication } from "interfaces/config";
import { IFormErrors } from "hooks/useFormValidation";

import isURL from "validator/lib/isURL";

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
    idp_name: config?.idp_name?.trim() || "",
    entity_id: config?.entity_id?.trim() || "",
    metadata_url: config?.metadata_url?.trim() || "",
    metadata: config?.metadata?.trim() || "",
  };
};

export const isEmptyFormData = (data: IFormDataIdp) => {
  return (
    !data.idp_name.trim() &&
    !data.entity_id.trim() &&
    !data.metadata.trim() &&
    !data.metadata_url.trim()
  );
};

const trimFormDataIdp = (data: IFormDataIdp): IFormDataIdp => ({
  idp_name: data.idp_name.trim(),
  entity_id: data.entity_id.trim(),
  metadata_url: data.metadata_url.trim(),
  metadata: data.metadata.trim(),
});

/**
 * Either field satisfies the metadata requirement, so they share one error
 * message. Only `metadata_url` can also hold a URL-format error.
 */
export const METADATA_SIBLING: Partial<
  Record<keyof IFormDataIdp, keyof IFormDataIdp>
> = {
  metadata: "metadata_url",
  metadata_url: "metadata",
};

export const validateEndUserAuthForm = (
  formData: IFormDataIdp
): IFormErrors => {
  const errors: IFormErrors = {};
  const data = trimFormDataIdp(formData);

  // An entirely empty form is how an admin clears the configuration, so it is
  // valid — required fields only apply once one of them has a value.
  if (isEmptyFormData(data)) {
    return errors;
  }

  if (!data.idp_name) {
    errors.idp_name = "Enter an identity provider name";
  }

  if (!data.entity_id) {
    errors.entity_id = "Enter an entity ID";
  }

  if (!data.metadata && !data.metadata_url) {
    errors.metadata = "Enter metadata or a metadata URL";
    errors.metadata_url = "Enter metadata or a metadata URL";
  } else if (data.metadata_url) {
    if (!isURL(data.metadata_url)) {
      errors.metadata_url = "Enter a valid metadata URL";
    } else if (
      !isURL(data.metadata_url, {
        require_protocol: true,
        protocols: ["http", "https"],
      })
    ) {
      errors.metadata_url =
        "Enter a metadata URL starting with https:// or http://";
    }
  }

  return errors;
};

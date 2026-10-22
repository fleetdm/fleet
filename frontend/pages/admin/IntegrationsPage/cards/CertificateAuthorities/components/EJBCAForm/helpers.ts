import { isValidURL } from "components/forms/validators";
import { IFormErrors } from "hooks/useFormValidation";
import {
  ICertificateAuthorityPartial,
  ICertificatesEJBCA,
} from "interfaces/certificates";
import { splitServerErrors } from "interfaces/errors";

import type { IEJBCAFormData } from "./EJBCAForm";

/** The server rejects a PKCS#12 over 1 MiB. */
export const MAX_P12_SIZE_BYTES = 1024 * 1024;

/** Where the server reports an error for each field of the form. */
const SERVER_ERROR_NAMES: Record<keyof IEJBCAFormData, string> = {
  name: "name",
  url: "url",
  clientP12: "client_p12",
  password: "password",
  trustCABundle: "trust_ca_bundle",
  caName: "certificate_authority_name",
  certificateProfileName: "certificate_profile_name",
  endEntityProfileName: "end_entity_profile_name",
  commonName: "certificate_common_name",
  usernameTemplate: "username_template",
  userPrincipalName: "certificate_user_principal_names",
};

interface IValidationContext {
  certAuthorities: ICertificateAuthorityPartial[];
  /** The id of the CA being edited, which doesn't conflict with itself. */
  editingId?: number;
}

export const validateEJBCAForm = (
  formData: IEJBCAFormData,
  { certAuthorities, editingId }: IValidationContext
): IFormErrors => {
  const errors: IFormErrors = {};
  const name = formData.name.trim();
  const isEditing = editingId !== undefined;

  if (!name) {
    errors.name = "Enter a name";
  } else if (!/^[a-zA-Z0-9_]+$/.test(name)) {
    errors.name = "Enter a name with only letters, numbers, and underscores";
  } else if (name.toLowerCase() === "ndes") {
    // The name of the NDES CA is fixed, and an EJBCA CA can't share its name with another CA.
    errors.name = "Enter a name other than NDES";
  } else if (
    certAuthorities.some(
      (ca) =>
        ca.id !== editingId && ca.name.toLowerCase() === name.toLowerCase()
    )
  ) {
    errors.name =
      "Enter a name that isn't used by another certificate authority";
  }

  const url = formData.url.trim();
  if (!url) {
    errors.url = "Enter a URL";
  } else if (!isValidURL({ url })) {
    errors.url = "Enter a valid URL";
  }

  // A new client certificate is needed when adding, and when editing it is optional, but then comes with its password.
  if (!isEditing && !formData.clientP12) {
    errors.clientP12 = "Upload a client certificate (.p12)";
  }
  if (formData.clientP12 && !formData.password) {
    errors.password = "Enter the password of the client certificate";
  }

  const bundle = formData.trustCABundle.trim();
  if (bundle && !bundle.includes("-----BEGIN CERTIFICATE-----")) {
    errors.trustCABundle = "Enter a PEM-encoded certificate";
  }

  if (!formData.caName.trim()) {
    errors.caName = "Enter an EJBCA CA name";
  }
  if (!formData.certificateProfileName.trim()) {
    errors.certificateProfileName = "Enter a certificate profile name";
  }
  if (!formData.endEntityProfileName.trim()) {
    errors.endEntityProfileName = "Enter an end entity profile name";
  }
  if (!formData.commonName.trim()) {
    errors.commonName = "Enter a common name";
  }

  return errors;
};

/** The form values of a stored EJBCA CA. The client certificate and its password aren't returned. */
export const generateEJBCAFormData = (
  certAuthority: ICertificatesEJBCA
): IEJBCAFormData => ({
  name: certAuthority.name,
  url: certAuthority.url,
  clientP12: "",
  password: "",
  trustCABundle: certAuthority.trust_ca_bundle ?? "",
  caName: certAuthority.certificate_authority_name,
  certificateProfileName: certAuthority.certificate_profile_name,
  endEntityProfileName: certAuthority.end_entity_profile_name,
  commonName: certAuthority.certificate_common_name,
  usernameTemplate: certAuthority.username_template ?? "",
  userPrincipalName: certAuthority.certificate_user_principal_names?.[0] ?? "",
});

export const generateEJBCAAddData = (
  formData: IEJBCAFormData
): ICertificatesEJBCA => ({
  name: formData.name,
  url: formData.url,
  client_p12: formData.clientP12,
  password: formData.password,
  ...(formData.trustCABundle && { trust_ca_bundle: formData.trustCABundle }),
  certificate_authority_name: formData.caName,
  certificate_profile_name: formData.certificateProfileName,
  end_entity_profile_name: formData.endEntityProfileName,
  certificate_common_name: formData.commonName,
  ...(formData.usernameTemplate && {
    username_template: formData.usernameTemplate,
  }),
  certificate_user_principal_names: formData.userPrincipalName
    ? [formData.userPrincipalName]
    : [],
});

/**
 * The fields of an EJBCA CA that were changed. It is empty if nothing was. The server checks the
 * connection to EJBCA whenever a connection field is sent, so a field that is the same is left out.
 */
export const generateEJBCAEditData = (
  certAuthority: ICertificatesEJBCA,
  formData: IEJBCAFormData
): Partial<ICertificatesEJBCA> => {
  const stored = generateEJBCAFormData(certAuthority);
  const changes: Partial<ICertificatesEJBCA> = {};
  if (formData.name !== stored.name) changes.name = formData.name;
  if (formData.url !== stored.url) changes.url = formData.url;
  if (formData.trustCABundle !== stored.trustCABundle) {
    changes.trust_ca_bundle = formData.trustCABundle;
  }
  if (formData.caName !== stored.caName) {
    changes.certificate_authority_name = formData.caName;
  }
  if (formData.certificateProfileName !== stored.certificateProfileName) {
    changes.certificate_profile_name = formData.certificateProfileName;
  }
  if (formData.endEntityProfileName !== stored.endEntityProfileName) {
    changes.end_entity_profile_name = formData.endEntityProfileName;
  }
  if (formData.commonName !== stored.commonName) {
    changes.certificate_common_name = formData.commonName;
  }
  if (formData.usernameTemplate !== stored.usernameTemplate) {
    changes.username_template = formData.usernameTemplate;
  }
  if (formData.userPrincipalName !== stored.userPrincipalName) {
    changes.certificate_user_principal_names = formData.userPrincipalName
      ? [formData.userPrincipalName]
      : [];
  }
  if (formData.clientP12) {
    changes.client_p12 = formData.clientP12;
    changes.password = formData.password;
  }
  return changes;
};

/** Reads a file as base64, which is how the API takes a PKCS#12. */
export const readFileAsBase64 = (file: File): Promise<string> => {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener("load", () => {
      // The result is a data URL: "data:<type>;base64,<data>".
      resolve(String(reader.result).split(",")[1] ?? "");
    });
    reader.addEventListener("error", () => reject(reader.error));
    reader.readAsDataURL(file);
  });
};

/**
 * The server starts its messages with what failed ("Couldn't add certificate authority.") and ends
 * them with a period. The form shows them next to a field, where that is not needed.
 */
const cleanFieldReason = (reason: string) =>
  reason
    .replace(/^Couldn't (add|edit) certificate authority\.\s*/, "")
    .replace(/\.$/, "");

/** Sorts the errors of a failed request into the ones for a field of the form, and the rest. */
export const splitEJBCAServerErrors = (err: unknown) => {
  const { fieldErrors, otherReasons } = splitServerErrors(
    err,
    SERVER_ERROR_NAMES
  );
  const errors: IFormErrors = {};
  Object.entries(fieldErrors).forEach(([field, reason]) => {
    if (reason) errors[field] = cleanFieldReason(reason);
  });
  return { fieldErrors: errors, otherReasons };
};

/**
 * The toast for the errors of a failed request that don't belong to a field. It falls back to the
 * message the server sent for the whole request, which starts with what failed.
 */
export const getEJBCAErrorMessage = (
  action: "add" | "edit",
  otherReasons: string[]
) => {
  const reasons = otherReasons.map(cleanFieldReason).filter(Boolean);
  return `Couldn't ${action} certificate authority. ${
    reasons.length ? `${reasons.join(". ")}.` : "Please try again."
  }`;
};

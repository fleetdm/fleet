import { ICertificateAuthorityPartial } from "interfaces/certificates";

import valid_url from "components/forms/validators/valid_url";

import { IEJBCAFormData } from "./EJBCAForm";

export interface IEJBCAFormValidation {
  isValid: boolean;
  name?: { isValid: boolean; message?: string };
  url?: { isValid: boolean; message?: string };
  clientP12Base64?: { isValid: boolean; message?: string };
  clientP12Password?: { isValid: boolean };
  certificateAuthorityNameEJBCA?: { isValid: boolean };
  certificateProfileName?: { isValid: boolean };
  endEntityProfileName?: { isValid: boolean };
  usernameTemplate?: { isValid: boolean };
}

type IMessageFunc = (formData: IEJBCAFormData) => string;
type IValidationMessage = string | IMessageFunc;
type IFormValidationKey = keyof Omit<IEJBCAFormValidation, "isValid">;

interface IValidation {
  name: string;
  isValid: (formData: IEJBCAFormData) => boolean;
  message?: IValidationMessage;
}

type IFormValidations = Record<
  IFormValidationKey,
  { validations: IValidation[] }
>;

export const generateFormValidations = (
  certAuthorities: ICertificateAuthorityPartial[],
  isEditing: boolean
) => {
  const FORM_VALIDATIONS: IFormValidations = {
    name: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) => formData.name.length > 0,
        },
        {
          name: "invalidCharacters",
          isValid: (formData: IEJBCAFormData) =>
            /^[a-zA-Z0-9_]+$/.test(formData.name),
          message:
            "Invalid characters. Only letters, numbers and underscores allowed.",
        },
        {
          name: "unique",
          isValid: (formData: IEJBCAFormData) =>
            isEditing ||
            certAuthorities.find(
              (cert) => cert.type === "ejbca" && cert.name === formData.name
            ) === undefined,
          message: "Name is already used by another EJBCA CA.",
        },
      ],
    },
    url: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) => formData.url.length > 0,
        },
        {
          name: "validUrl",
          // EJBCA is self-hosted, so localhost / IPs are legitimate URLs
          // (dev setups, SSH tunnels, internal LBs without DNS). Unlike
          // DigiCert which is SaaS at a fixed public domain.
          isValid: (formData: IEJBCAFormData) =>
            valid_url({ url: formData.url, allowLocalHost: true }),
          message: "Must be a valid URL.",
        },
      ],
    },
    clientP12Base64: {
      validations: [
        {
          // When editing, the cert+key are already stored — uploading a new
          // P12 is optional and only triggers a rotation.
          name: "required",
          isValid: (formData: IEJBCAFormData) =>
            isEditing || formData.clientP12Base64.length > 0,
          message: "A PKCS#12 client certificate is required.",
        },
      ],
    },
    clientP12Password: {
      validations: [
        {
          // Password is only required when a new P12 has been supplied
          // (whether on create or during edit rotation).
          name: "required-with-p12",
          isValid: (formData: IEJBCAFormData) =>
            formData.clientP12Base64.length === 0 ||
            formData.clientP12Password.length > 0,
        },
      ],
    },
    certificateAuthorityNameEJBCA: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) =>
            formData.certificateAuthorityNameEJBCA.length > 0,
        },
      ],
    },
    certificateProfileName: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) =>
            formData.certificateProfileName.length > 0,
        },
      ],
    },
    endEntityProfileName: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) =>
            formData.endEntityProfileName.length > 0,
        },
      ],
    },
    usernameTemplate: {
      validations: [
        {
          name: "required",
          isValid: (formData: IEJBCAFormData) =>
            formData.usernameTemplate.length > 0,
        },
      ],
    },
  };
  return FORM_VALIDATIONS;
};

const getErrorMessage = (
  formData: IEJBCAFormData,
  message?: IValidationMessage
) => {
  if (message === undefined || typeof message === "string") {
    return message;
  }
  return message(formData);
};

export const validateFormData = (
  formData: IEJBCAFormData,
  validationConfig: IFormValidations
) => {
  const formValidation: IEJBCAFormValidation = { isValid: true };

  Object.keys(validationConfig).forEach((key) => {
    const objKey = key as keyof typeof validationConfig;
    const failedValidation = validationConfig[objKey].validations.find(
      (validation) => !validation.isValid(formData)
    );

    if (!failedValidation) {
      formValidation[objKey] = { isValid: true };
    } else {
      formValidation.isValid = false;
      formValidation[objKey] = {
        isValid: false,
        message: getErrorMessage(formData, failedValidation.message),
      };
    }
  });

  return formValidation;
};

/**
 * Read a File into a base64 string suitable for sending to the API as
 * `client_p12`. The server decodes the bytes with `client_p12_password`,
 * extracts cert + key, and discards both upload fields. Strips the
 * `data:...;base64,` prefix that FileReader prepends.
 */
export const readFileAsBase64 = (file: File): Promise<string> =>
  new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(reader.error);
    reader.onload = () => {
      const result = reader.result as string;
      const commaIdx = result.indexOf(",");
      resolve(commaIdx === -1 ? result : result.slice(commaIdx + 1));
    };
    reader.readAsDataURL(file);
  });

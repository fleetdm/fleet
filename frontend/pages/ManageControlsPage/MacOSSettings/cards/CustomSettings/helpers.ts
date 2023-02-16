import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

export const ERROR_MESSAGES = {
  wrongType: {
    condition: () => false,
    message: "Couldn’t upload. The file should be a .mobileconfig file.",
  },
  identifierExists: {
    condition: (message: string) => message.includes("PayloadIdentifier"),
    message:
      "Couldn’t upload. A configuration profile with this identifier (PayloadIdentifier) already exists.",
  },
  nameExists: {
    condition: (message: string) => message.includes("PayloadDisplayName"),
    message:
      "Couldn’t upload. A configuration profile with this name (PayloadDisplayName) already exists.",
  },
  encrypted: {
    condition: (message: string) => message.includes("unencrypted"),
    message: "Couldn’t upload. The file should be unencrypted.",
  },
  validXML: {
    condition: (message: string) => message.includes("valid XML"),
    message: "Couldn’t upload. The file should include valid XML.",
  },
  fileVault: {
    condition: (message: string) =>
      message.includes("unsupported PayloadType(s): com.apple.MCX.FileVault2"),
    message:
      "Couldn’t upload. The configuration profile can’t include FileVault settings. To control these settings, go to Disk encryption.",
  },
  default: {
    condition: () => false,
    message: "Couldn’t upload. Please try again.",
  },
};

export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiMessage = err.data.message;

  const error = Object.values(ERROR_MESSAGES).find(
    (errType) => errType?.condition && errType.condition(apiMessage)
  );

  if (!error) {
    return ERROR_MESSAGES.default.message;
  }

  return error.message;
};

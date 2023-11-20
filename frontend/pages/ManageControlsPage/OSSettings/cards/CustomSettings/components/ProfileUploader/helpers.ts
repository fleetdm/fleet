import { AxiosResponse } from "axios";
import { IApiError } from "interfaces/errors";

export const UPLOAD_ERROR_MESSAGES = {
  wrongType: {
    condition: () => false,
    message: "Couldn’t upload. The file should be a .mobileconfig file.",
  },
  identifierExists: {
    condition: (reason: string) =>
      reason.includes("MDMAppleConfigProfile.PayloadIdentifier"),
    message:
      "Couldn’t upload. A configuration profile with this identifier (PayloadIdentifier) already exists.",
  },
  nameExists: {
    condition: (reason: string) => reason.includes("PayloadDisplayName"),
    message:
      "Couldn’t upload. A configuration profile with this name (PayloadDisplayName) already exists.",
  },
  encrypted: {
    condition: (reason: string) => reason.includes("encrypted"),
    message: "Couldn’t upload. The file should be unencrypted.",
  },
  validXML: {
    condition: (reason: string) => reason.includes("parsing XML"),
    message: "Couldn’t upload. The file should include valid XML.",
  },
  fileVault: {
    condition: (reason: string) =>
      reason.includes("unsupported PayloadType(s): com.apple.MCX.FileVault2"),
    message:
      "Couldn’t upload. The configuration profile can’t include FileVault settings. To control these settings, go to Disk encryption.",
  },
  default: {
    condition: () => false,
    message: "Couldn’t upload. Please try again.",
  },
};

export const getErrorMessage = (err: AxiosResponse<IApiError>) => {
  const apiReason = err.data.errors[0].reason;

  const error = Object.values(UPLOAD_ERROR_MESSAGES).find((errType) =>
    errType.condition(apiReason)
  );

  if (!error) {
    return UPLOAD_ERROR_MESSAGES.default.message;
  }

  return error.message;
};

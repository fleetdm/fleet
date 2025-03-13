import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  isCustomSCEPCertIntegration,
  isDigicertCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";

import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { get } from "lodash";

const DEFAULT_ERROR_MESSAGE =
  "Couldn't edit certificate authority. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const generateErrorMessage = (e: unknown) => {
  return DEFAULT_ERROR_MESSAGE;
};

export const generateDefaultFormData = (
  certAuthority: ICertificateIntegration
): IDigicertFormData | null => {
  if (isDigicertCertIntegration(certAuthority)) {
    return {
      name: certAuthority.name,
      url: certAuthority.url,
      apiToken: certAuthority.api_token,
      profileId: certAuthority.profile_id,
      commonName: certAuthority.certificate_common_name,
      userPrincipalName: certAuthority.certificate_user_principal_names[0],
      certificateSeatId: certAuthority.certificate_seat_id,
    };
  }

  return null;
};

export const getCertificateAuthorityType = (
  certAuthority: ICertificateIntegration
): ICertificateAuthorityType => {
  if (isNDESCertIntegration(certAuthority)) return "ndes";
  if (isCustomSCEPCertIntegration(certAuthority)) return "custom";
  return "digicert";
};

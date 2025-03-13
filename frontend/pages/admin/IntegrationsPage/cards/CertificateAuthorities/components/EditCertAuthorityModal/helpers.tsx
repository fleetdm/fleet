import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationDigicert,
  isCustomSCEPCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";

const DEFAULT_ERROR_MESSAGE =
  "Couldn't edit certificate authority. Please try again.";

// eslint-disable-next-line import/prefer-default-export
export const generateErrorMessage = (e: unknown) => {
  return DEFAULT_ERROR_MESSAGE;
};

export const generateDefaultFormData = (
  certAuthority: ICertificateIntegration
): ICertFormData => {
  const cert = certAuthority as ICertificatesIntegrationDigicert;
  return {
    name: cert.name,
    url: cert.url,
    apiToken: cert.api_token,
    profileId: cert.profile_id,
    commonName: cert.certificate_common_name,
    userPrincipalName: cert.certificate_user_principal_names[0],
    certificateSeatId: cert.certificate_seat_id,
  };
};

export const getCertificateAuthorityType = (
  certAuthority: ICertificateIntegration
): ICertificateAuthorityType => {
  if (isNDESCertIntegration(certAuthority)) return "ndes";
  if (isCustomSCEPCertIntegration(certAuthority)) return "custom";
  return "digicert";
};

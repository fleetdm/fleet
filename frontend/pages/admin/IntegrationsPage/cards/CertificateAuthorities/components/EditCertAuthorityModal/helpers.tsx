import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationCustomSCEP,
  isCustomSCEPCertIntegration,
  isDigicertCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { getDisplayErrMessage } from "../AddCertAuthorityModal/helpers";

export const getCertificateAuthorityType = (
  certAuthority: ICertificateIntegration
): ICertificateAuthorityType => {
  if (isNDESCertIntegration(certAuthority)) return "ndes";
  if (isCustomSCEPCertIntegration(certAuthority)) return "custom";
  return "digicert";
};

export const generateDefaultFormData = (
  certAuthority: ICertificateIntegration
): ICertFormData => {
  if (isNDESCertIntegration(certAuthority)) {
    return {
      scepURL: certAuthority.url,
      adminURL: certAuthority.admin_url,
      username: certAuthority.username,
      password: certAuthority.password,
    };
  } else if (isDigicertCertIntegration(certAuthority)) {
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

  const customSCEPcert = certAuthority as ICertificatesIntegrationCustomSCEP;
  return {
    name: customSCEPcert.name,
    scepURL: customSCEPcert.url,
    challenge: customSCEPcert.challenge,
  };
};

export const updateFormData = (
  certAuthority: ICertificateIntegration,
  prevFormData: ICertFormData,
  update: { name: string; value: string }
) => {
  const newData = { ...prevFormData, [update.name]: update.value };

  // for some inputs that change we want to reset one of the other inputs
  // and force users to re-enter it.
  if (isDigicertCertIntegration(certAuthority)) {
    if (
      update.name === "name" ||
      update.name === "url" ||
      update.name === "profileId"
    ) {
      return {
        ...newData,
        apiToken: "",
      };
    }
  } else if (isNDESCertIntegration(certAuthority)) {
    if (update.name === "adminURL" || update.name === "username") {
      return {
        ...newData,
        password: "",
      };
    }
  } else if (isCustomSCEPCertIntegration(certAuthority)) {
    if (update.name === "name" || update.name === "scepURL") {
      return {
        ...newData,
        challenge: "",
      };
    }
  }

  return newData;
};

export const getErrorMessage = (err: unknown) => {
  return `Couldn't edit certificate authority. ${getDisplayErrMessage(err)}`;
};

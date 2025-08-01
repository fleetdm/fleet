import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationCustomSCEP,
  isCustomSCEPCertIntegration,
  isDigicertCertIntegration,
  isHydrantCertIntegration,
  isNDESCertIntegration,
} from "interfaces/integration";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { getDisplayErrMessage } from "../AddCertAuthorityModal/helpers";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";

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
      userPrincipalName:
        certAuthority.certificate_user_principal_names?.[0] ?? "",
      certificateSeatId: certAuthority.certificate_seat_id,
    };
  } else if (isHydrantCertIntegration(certAuthority)) {
    return {
      name: certAuthority.name,
      url: certAuthority.url,
      clientId: certAuthority.client_id,
      clientSecret: certAuthority.client_secret,
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
  // and force users to re-enter it. we only want to clear these values if it
  // has not been updated. The characters "********" is the value the API sends
  // back so we check for that value to determine if its been changed or not.
  if (isDigicertCertIntegration(certAuthority)) {
    const formData = prevFormData as IDigicertFormData;
    if (
      update.name === "name" ||
      update.name === "url" ||
      update.name === "profileId"
    ) {
      return {
        ...newData,
        apiToken: formData.apiToken === "********" ? "" : formData.apiToken,
      };
    }
  } else if (isNDESCertIntegration(certAuthority)) {
    const formData = prevFormData as INDESFormData;
    if (update.name === "adminURL" || update.name === "username") {
      return {
        ...newData,
        password: formData.password === "********" ? "" : formData.password,
      };
    }
  } else if (isCustomSCEPCertIntegration(certAuthority)) {
    const formData = prevFormData as ICustomSCEPFormData;
    if (update.name === "name" || update.name === "scepURL") {
      return {
        ...newData,
        challenge: formData.challenge === "********" ? "" : formData.challenge,
      };
    }
  } else if (isHydrantCertIntegration(certAuthority)) {
    const formData = prevFormData as IHydrantFormData;
    if (
      update.name === "name" ||
      update.name === "url" ||
      update.name === "clientId"
    ) {
      return {
        ...newData,
        clientSecret:
          formData.clientSecret === "********" ? "" : formData.clientSecret,
      };
    }
  }

  return newData;
};

export const getErrorMessage = (err: unknown) => {
  return `Couldn't edit certificate authority. ${getDisplayErrMessage(err)}`;
};

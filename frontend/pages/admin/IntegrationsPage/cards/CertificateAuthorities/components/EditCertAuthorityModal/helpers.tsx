import React from "react";

import { IEditCertAuthorityFormData } from "services/entities/certificates";
import {
  ICertificateAuthority,
  ICertificatesCustomSCEP,
} from "interfaces/certificates";
import deepDifference from "utilities/deep_difference";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { getDisplayErrMessage } from "../AddCertAuthorityModal/helpers";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { IEJBCAFormData } from "../EJBCAForm/EJBCAForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";
import { ISmallstepFormData } from "../SmallstepForm/SmallstepForm";
import { ICustomESTFormData } from "../CustomESTForm/CustomESTForm";

export const generateDefaultFormData = (
  certAuthority: ICertificateAuthority
): ICertFormData => {
  switch (certAuthority.type) {
    case "ndes_scep_proxy":
      return {
        scepURL: certAuthority.url,
        adminURL: certAuthority.admin_url,
        username: certAuthority.username,
        password: certAuthority.password,
      };
    case "digicert":
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
    case "hydrant":
      return {
        name: certAuthority.name,
        url: certAuthority.url,
        clientId: certAuthority.client_id,
        clientSecret: certAuthority.client_secret,
      };
    case "smallstep":
      return {
        name: certAuthority.name,
        scepURL: certAuthority.url,
        challengeURL: certAuthority.challenge_url,
        username: certAuthority.username,
        password: certAuthority.password,
      };
    case "custom_scep_proxy": {
      const customSCEPcert = certAuthority as ICertificatesCustomSCEP;
      return {
        name: customSCEPcert.name,
        scepURL: customSCEPcert.url,
        challenge: customSCEPcert.challenge,
      };
    }
    case "custom_est_proxy":
      return {
        name: certAuthority.name,
        url: certAuthority.url,
        username: certAuthority.username,
        password: certAuthority.password,
      };
    case "ejbca":
      return {
        name: certAuthority.name,
        url: certAuthority.url,
        // Edit form leaves the upload fields empty unless the admin uploads
        // a new .p12 to rotate.
        clientP12Base64: "",
        clientP12FileName: "",
        clientP12Password: "",
        trustCABundle: certAuthority.trust_ca_bundle ?? "",
        certificateAuthorityNameEJBCA:
          certAuthority.certificate_authority_name_ejbca,
        certificateProfileName: certAuthority.certificate_profile_name,
        endEntityProfileName: certAuthority.end_entity_profile_name,
        usernameTemplate: certAuthority.username_template,
        userPrincipalName:
          certAuthority.certificate_user_principal_names?.[0] ?? "",
      };
    default:
      throw new Error(
        `Unknown certificate authority type: ${certAuthority.type}`
      );
  }
};

export const generateEditCertAuthorityData = (
  certAuthority: ICertificateAuthority,
  formData: ICertFormData
): IEditCertAuthorityFormData => {
  const certAuthWithoutType = Object.assign({}, certAuthority);
  delete certAuthWithoutType.type;
  delete certAuthWithoutType.id;

  switch (certAuthority.type) {
    case "ndes_scep_proxy": {
      const {
        scepURL,
        adminURL,
        username,
        password,
      } = formData as INDESFormData;
      return {
        ndes_scep_proxy: deepDifference(
          {
            url: scepURL,
            admin_url: adminURL,
            username,
            password,
          },
          certAuthWithoutType
        ),
      };
    }
    case "digicert": {
      const {
        name,
        url: digicertUrl,
        apiToken,
        profileId,
        commonName,
        userPrincipalName,
        certificateSeatId,
      } = formData as IDigicertFormData;
      return {
        digicert: deepDifference(
          {
            name,
            url: digicertUrl,
            api_token: apiToken,
            profile_id: profileId,
            certificate_common_name: commonName,
            certificate_user_principal_names: [userPrincipalName],
            certificate_seat_id: certificateSeatId,
          },
          certAuthWithoutType
        ),
      };
    }
    case "hydrant": {
      const {
        name: hydrantName,
        url: hydrantUrl,
        clientId,
        clientSecret,
      } = formData as IHydrantFormData;
      return {
        hydrant: deepDifference(
          {
            name: hydrantName,
            url: hydrantUrl,
            client_id: clientId,
            client_secret: clientSecret,
          },
          certAuthWithoutType
        ),
      };
    }
    case "smallstep": {
      const {
        name: smallstepName,
        scepURL: smallstepURL,
        challengeURL: smallstepChallengeURL,
        username: smallstepUsername,
        password: smallstepPassword,
      } = formData as ISmallstepFormData;
      return {
        smallstep: deepDifference(
          {
            name: smallstepName,
            url: smallstepURL,
            challenge_url: smallstepChallengeURL,
            username: smallstepUsername,
            password: smallstepPassword,
          },
          certAuthWithoutType
        ),
      };
    }
    case "custom_scep_proxy": {
      const {
        name: customSCEPName,
        scepURL: customSCEPUrl,
        challenge,
      } = formData as ICustomSCEPFormData;
      return {
        custom_scep_proxy: deepDifference(
          {
            name: customSCEPName,
            url: customSCEPUrl,
            challenge,
          },
          certAuthWithoutType
        ),
      };
    }
    case "custom_est_proxy": {
      const {
        name: customESTName,
        url: customESTUrl,
        username: customESTUsername,
        password: customESTPassword,
      } = formData as ICustomESTFormData;
      const diff = {
        custom_est_proxy: deepDifference(
          {
            name: customESTName,
            url: customESTUrl,
            username: customESTUsername,
            password: customESTPassword,
          },
          certAuthWithoutType
        ),
      };
      // Make sure credentials are included if we are modifying the url
      if (diff.custom_est_proxy.url) {
        if (!diff.custom_est_proxy.username) {
          diff.custom_est_proxy.username = customESTUsername;
        }
        if (!diff.custom_est_proxy.password) {
          diff.custom_est_proxy.password = customESTPassword;
        }
      }
      return diff;
    }
    case "ejbca": {
      const {
        name: ejbcaName,
        url: ejbcaUrl,
        clientP12Base64,
        clientP12Password,
        trustCABundle,
        certificateAuthorityNameEJBCA,
        certificateProfileName,
        endEntityProfileName,
        usernameTemplate,
        userPrincipalName,
      } = formData as IEJBCAFormData;
      // Compute the diff against the stored EJBCA fields, then inject the
      // P12 + password only when the admin actually uploaded a new bundle
      // (rotation). Both must travel together — the backend rejects one
      // without the other.
      const diff = deepDifference(
        {
          name: ejbcaName,
          url: ejbcaUrl,
          trust_ca_bundle: trustCABundle,
          certificate_authority_name_ejbca: certificateAuthorityNameEJBCA,
          certificate_profile_name: certificateProfileName,
          end_entity_profile_name: endEntityProfileName,
          username_template: usernameTemplate,
          certificate_user_principal_names: userPrincipalName
            ? [userPrincipalName]
            : null,
        },
        certAuthWithoutType
      ) as Record<string, unknown>;
      if (clientP12Base64) {
        diff.client_p12 = clientP12Base64;
        diff.client_p12_password = clientP12Password;
      }
      return { ejbca: diff };
    }
    default:
      throw new Error(
        `Unknown certificate authority type: ${certAuthority.type}`
      );
  }
};

export const updateFormData = (
  certAuthority: ICertificateAuthority,
  prevFormData: ICertFormData,
  update: { name: string; value: string }
) => {
  const newData = { ...prevFormData, [update.name]: update.value };

  // for some inputs that change we want to reset one or more of the other inputs
  // and force users to re-enter them. we only want to clear these values if it
  // has not been updated. The characters "********" is the value the API sends
  // back so we check for that value to determine if it's been changed or not.
  switch (certAuthority.type) {
    case "digicert": {
      const formData = prevFormData as IDigicertFormData;
      if (
        update.name === "name" ||
        update.name === "url" ||
        update.name === "profileId"
      ) {
        return {
          ...newData,
          apiToken:
            formData.apiToken === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.apiToken,
        };
      }
      break;
    }
    case "ndes_scep_proxy": {
      const formData = prevFormData as INDESFormData;
      if (update.name === "adminURL" || update.name === "username") {
        return {
          ...newData,
          password:
            formData.password === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.password,
        };
      }
      break;
    }
    case "custom_scep_proxy": {
      const formData = prevFormData as ICustomSCEPFormData;
      if (update.name === "name" || update.name === "scepURL") {
        return {
          ...newData,
          challenge:
            formData.challenge === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.challenge,
        };
      }
      break;
    }
    case "hydrant": {
      // for Hydrant, we reset clientId and clientSecret if name or url changes
      // and the fields have not been updated. We do this to force users to send
      // the correct clientId and clientSecret for the new name or url.
      const formData = prevFormData as IHydrantFormData;
      if (update.name === "name" || update.name === "url") {
        return {
          ...newData,
          clientId:
            formData.clientId === certAuthority.client_id
              ? ""
              : formData.clientId,
          clientSecret:
            formData.clientSecret === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.clientSecret,
        };
      }
      break;
    }
    case "smallstep": {
      const formData = prevFormData as ISmallstepFormData;
      if (
        update.name === "name" ||
        update.name === "scepURL" ||
        update.name === "challengeURL" ||
        update.name === "username"
      ) {
        return {
          ...newData,
          password:
            formData.password === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.password,
        };
      }
      break;
    }
    case "custom_est_proxy": {
      const formData = prevFormData as ICustomESTFormData;
      if (update.name === "url") {
        return {
          ...newData,
          username:
            formData.username === certAuthority.username
              ? ""
              : formData.username,
          password:
            formData.password === UNCHANGED_PASSWORD_API_RESPONSE
              ? ""
              : formData.password,
        };
      }
      break;
    }
    case "ejbca":
      // No special "clear other fields" behavior for EJBCA edits. The
      // rotation case (new P12 + password) is handled by the form
      // itself — both fields travel together when present, and neither
      // is auto-cleared by editing a different field.
      break;
    default:
      throw new Error(
        `Unknown certificate authority type: ${certAuthority.type}`
      );
  }
  return newData;
};

export const getErrorMessage = (err: unknown): JSX.Element => {
  return (
    <>Couldn&apos;t edit certificate authority. {getDisplayErrMessage(err)}</>
  );
};

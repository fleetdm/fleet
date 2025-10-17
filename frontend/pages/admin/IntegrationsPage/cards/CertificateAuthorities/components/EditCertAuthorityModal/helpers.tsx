import React from "react";

import { IEditCertAuthorityBody } from "services/entities/certificates";
import {
  ICertificateAuthority,
  ICertificatesCustomSCEP,
} from "interfaces/certificates";
import deepDifference from "utilities/deep_difference";

import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { getDisplayErrMessage } from "../AddCertAuthorityModal/helpers";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";
import { ISmallstepFormData } from "../SmallstepForm/SmallstepForm";

export const generateDefaultFormData = (
  certAuthority: ICertificateAuthority
): ICertFormData => {
  if (certAuthority.type === "ndes_scep_proxy") {
    return {
      scepURL: certAuthority.url,
      adminURL: certAuthority.admin_url,
      username: certAuthority.username,
      password: certAuthority.password,
    };
  } else if (certAuthority.type === "digicert") {
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
  } else if (certAuthority.type === "hydrant") {
    return {
      name: certAuthority.name,
      url: certAuthority.url,
      clientId: certAuthority.client_id,
      clientSecret: certAuthority.client_secret,
    };
  } else if (certAuthority.type === "smallstep") {
    return {
      name: certAuthority.name,
      scepURL: certAuthority.url,
      challengeURL: certAuthority.challenge_url,
      username: certAuthority.username,
      password: certAuthority.password,
    };
  }

  // FIXME: seems like we have some competing patterns in here where we sometimes do switch
  // statements with a default and sometimes do if or if/else if with a final default return. We
  // should probably standardize on one or the other. Also, do we really want this to be the
  // default? Why not have an explicit check for custom_scep_proxy and have the final
  // else throw an error?

  const customSCEPcert = certAuthority as ICertificatesCustomSCEP;
  return {
    name: customSCEPcert.name,
    scepURL: customSCEPcert.url,
    challenge: customSCEPcert.challenge,
  };
};

export const generateEditCertAuthorityData = (
  certAuthority: ICertificateAuthority,
  formData: ICertFormData
): IEditCertAuthorityBody => {
  const certAuthWithoutType = Object.assign({}, certAuthority);
  delete certAuthWithoutType.type;
  delete certAuthWithoutType.id;

  switch (certAuthority.type) {
    case "ndes_scep_proxy":
      // eslint-disable-next-line no-case-declarations
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
    case "digicert":
      // eslint-disable-next-line no-case-declarations
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
    case "hydrant":
      // eslint-disable-next-line no-case-declarations
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
    case "smallstep":
      // eslint-disable-next-line no-case-declarations
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

    // FIXME: do we really want this to be the default? why not have an explicit case for
    // custom_scep_proxy and have the default throw an error?
    default:
      // custom_scep_proxy
      // eslint-disable-next-line no-case-declarations
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
};

export const updateFormData = (
  certAuthority: ICertificateAuthority,
  prevFormData: ICertFormData,
  update: { name: string; value: string }
) => {
  const newData = { ...prevFormData, [update.name]: update.value };

  // for some inputs that change we want to reset one of the other inputs
  // and force users to re-enter it. we only want to clear these values if it
  // has not been updated. The characters "********" is the value the API sends
  // back so we check for that value to determine if its been changed or not.
  if (certAuthority.type === "digicert") {
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
  } else if (certAuthority.type === "ndes_scep_proxy") {
    const formData = prevFormData as INDESFormData;
    if (update.name === "adminURL" || update.name === "username") {
      return {
        ...newData,
        password: formData.password === "********" ? "" : formData.password,
      };
    }
  } else if (certAuthority.type === "custom_scep_proxy") {
    const formData = prevFormData as ICustomSCEPFormData;
    if (update.name === "name" || update.name === "scepURL") {
      return {
        ...newData,
        challenge: formData.challenge === "********" ? "" : formData.challenge,
      };
    }
  } else if (certAuthority.type === "hydrant") {
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
          formData.clientSecret === "********" ? "" : formData.clientSecret,
      };
    }
  } else if (certAuthority.type === "smallstep") {
    const formData = prevFormData as ISmallstepFormData;
    if (
      update.name === "name" ||
      update.name === "scepURL" ||
      update.name === "challengeURL" ||
      update.name === "username"
    ) {
      return {
        ...newData,
        password: formData.password === "********" ? "" : formData.password,
      };
    }
  }

  return newData;
};

export const getErrorMessage = (err: unknown): JSX.Element => {
  return (
    <>Couldn&apos;t edit certificate authority. {getDisplayErrMessage(err)}</>
  );
};

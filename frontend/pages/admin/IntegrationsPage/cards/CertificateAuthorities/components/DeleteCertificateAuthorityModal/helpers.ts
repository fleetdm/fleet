import { useCallback, useContext } from "react";

import { IAddCertAuthorityBody } from "services/entities/certificates";
import { AppContext } from "context/app";

import {
  ICertificateAuthorityType,
} from "interfaces/certificates";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";
import { IHydrantFormData } from "../HydrantForm/HydrantForm";

/**
 * Generates the data to be sent to the API to add a new certificate authority.
 * This function constructs the request body based on the selected certificate authority type
 * and the provided form data.
 */
export const generateAddCertAuthorityData = (
  certAuthorityType: ICertificateAuthorityType,
  formData: ICertFormData
): IAddCertAuthorityBody | undefined => {
  switch (certAuthorityType) {
    case "ndes_scep_proxy":
      // eslint-disable-next-line no-case-declarations
      const {
        scepURL,
        adminURL,
        username,
        password,
      } = formData as INDESFormData;
      return {
        ndes_scep_proxy: {
          url: scepURL,
          admin_url: adminURL,
          username,
          password,
        },
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
        digicert: {
          name,
          url: digicertUrl,
          api_token: apiToken,
          profile_id: profileId,
          certificate_common_name: commonName,
          certificate_user_principal_names: [userPrincipalName],
          certificate_seat_id: certificateSeatId,
        },
      };
    case "custom_scep_proxy":
      // eslint-disable-next-line no-case-declarations
      const {
        name: customSCEPName,
        scepURL: customSCEPUrl,
        challenge,
      } = formData as ICustomSCEPFormData;
      return {
        custom_scep_proxy: {
          name: customSCEPName,
          url: customSCEPUrl,
          challenge,
        },
      };
    case "hydrant":
      // eslint-disable-next-line no-case-declarations
      const { name: hydrantName, url, clientId, clientSecret } = formData as IHydrantFormData;
      return {
        hydrant: {
          name: hydrantName,
          url,
          client_id: clientId,
          client_secret: clientSecret,
        },
      };
    default:
      return undefined;
};

export const useCertAuthorityDataGenerator = (
  certAuthorityType: ICertificateAuthorityType,
  certAuthority?: ICertificateIntegration
) => {
  const { config } = useContext(AppContext);

  /**
   * generates the data to be sent to the API to edit the certificate authority.
   * under the hood we are updating the app config object with the new data and
   * have to generate the correct data for the PATCH request.
   */
  const generateEditPatchData = useCallback(
    (formData: ICertFormData) => {
      if (!config) return null;

      const data: { integrations: Partial<IGlobalIntegrations> } = {
        integrations: {},
      };

      switch (certAuthorityType) {
        case "ndes":
          // eslint-disable-next-line no-case-declarations
          const {
            scepURL: ndesSCEPUrl,
            adminURL,
            username,
            password,
          } = formData as INDESFormData;
          data.integrations.ndes_scep_proxy = {
            url: ndesSCEPUrl,
            admin_url: adminURL,
            username,
            password,
          };
          break;
        case "digicert":
          // eslint-disable-next-line no-case-declarations
          const {
            name: digicertName,
            url,
            apiToken,
            profileId,
            commonName,
            userPrincipalName,
            certificateSeatId,
          } = formData as IDigicertFormData;
          data.integrations.digicert = config.integrations.digicert?.map(
            (cert) => {
              // only update the certificate authority that we are editing
              if (
                (certAuthority as ICertificatesIntegrationDigicert).name ===
                cert.name
              ) {
                return {
                  name: digicertName,
                  url,
                  api_token: apiToken,
                  profile_id: profileId,
                  certificate_common_name: commonName,
                  certificate_user_principal_names:
                    userPrincipalName !== "" ? [userPrincipalName] : [],
                  certificate_seat_id: certificateSeatId,
                };
              }
              return cert;
            }
          );
          break;
        case "custom":
          // eslint-disable-next-line no-case-declarations
          const {
            name: customSCEPName,
            scepURL: customSCEPUrl,
            challenge,
          } = formData as ICustomSCEPFormData;
          data.integrations.custom_scep_proxy = config.integrations.custom_scep_proxy?.map(
            (cert) => {
              // only update the certificate authority that we are editing
              if (
                (certAuthority as ICertificatesIntegrationCustomSCEP).name ===
                cert.name
              ) {
                return {
                  name: customSCEPName,
                  url: customSCEPUrl,
                  challenge,
                };
              }
              return cert;
            }
          );
          break;
        default:
          break;
      }

      return data;
    },
    [certAuthority, certAuthorityType, config]
  );

  return {
    generateEditPatchData,
  };
};

export const generateCertAuthorityDisplayName = (
  certAuthorityType: ICertificateAuthorityType,
  certAuthority: ICertificateIntegration
) => {
  switch (certAuthorityType) {
    case "ndes":
      return "NDES";
    case "digicert":
      return (certAuthority as ICertificatesIntegrationDigicert).name;
    case "custom":
      return (certAuthority as ICertificatesIntegrationCustomSCEP).name;
    case "hydrant":
      return (certAuthority as ICertificatesIntegrationHydrant).name;
    default:
      return "";
  }
};

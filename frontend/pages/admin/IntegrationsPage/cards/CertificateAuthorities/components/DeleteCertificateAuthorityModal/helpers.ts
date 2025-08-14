import { AppContext } from "context/app";

import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationCustomSCEP,
  ICertificatesIntegrationDigicert,
  ICertificatesIntegrationHydrant,
  IGlobalIntegrations,
} from "interfaces/integration";
import { useCallback, useContext } from "react";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";
import { ICertFormData } from "../AddCertAuthorityModal/AddCertAuthorityModal";
import { INDESFormData } from "../NDESForm/NDESForm";
import { ICustomSCEPFormData } from "../CustomSCEPForm/CustomSCEPForm";

export const useCertAuthorityDataGenerator = (
  certAuthorityType: ICertificateAuthorityType,
  certAuthority?: ICertificateIntegration
) => {
  const { config } = useContext(AppContext);

  const generateAddPatchData = useCallback(
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
          data.integrations.digicert = [
            ...(config.integrations.digicert || []),
            {
              name: digicertName,
              url,
              api_token: apiToken,
              profile_id: profileId,
              certificate_common_name: commonName,
              certificate_user_principal_names: [userPrincipalName],
              certificate_seat_id: certificateSeatId,
            },
          ];
          break;
        case "custom":
          // eslint-disable-next-line no-case-declarations
          const {
            name: customSCEPName,
            scepURL: customSCEPUrl,
            challenge,
          } = formData as ICustomSCEPFormData;
          data.integrations.custom_scep_proxy = [
            ...(config.integrations.custom_scep_proxy || []),
            {
              name: customSCEPName,
              url: customSCEPUrl,
              challenge,
            },
          ];
          break;
        default:
          break;
      }

      return data;
    },
    [certAuthorityType, config]
  );

  /**
   * generates the data to be sent to the API to delete the certificate authority.
   * under the hood we are updating the app config object with the new data and
   * have to generate the correct data for the PATCH request.
   */
  const generateDeletePatchData = useCallback(() => {
    if (!config) return null;

    const data: { integrations: Partial<IGlobalIntegrations> } = {
      integrations: {},
    };

    switch (certAuthorityType) {
      case "ndes":
        data.integrations.ndes_scep_proxy = null;
        break;
      case "digicert":
        data.integrations.digicert = config.integrations.digicert?.filter(
          (cert) => {
            return (
              (certAuthority as ICertificatesIntegrationDigicert).name !==
              cert.name
            );
          }
        );
        break;
      case "custom":
        data.integrations.custom_scep_proxy = config.integrations.custom_scep_proxy?.filter(
          (cert) => {
            return (
              (certAuthority as ICertificatesIntegrationCustomSCEP).name !==
              cert.name
            );
          }
        );
        break;
      default:
        break;
    }

    return data;
  }, [certAuthority, certAuthorityType, config]);

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
    generateAddPatchData,
    generateDeletePatchData,
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

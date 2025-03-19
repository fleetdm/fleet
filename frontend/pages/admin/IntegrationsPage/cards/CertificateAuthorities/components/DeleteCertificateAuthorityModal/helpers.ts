import { AppContext } from "context/app";

import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationCustomSCEP,
  ICertificatesIntegrationDigicert,
  IGlobalIntegrations,
} from "interfaces/integration";
import { useCallback, useContext } from "react";
import { IDigicertFormData } from "../DigicertForm/DigicertForm";

export const useCertAuthorityDataGenerator = (
  certAuthorityType: ICertificateAuthorityType,
  certAuthority?: ICertificateIntegration
) => {
  const { config } = useContext(AppContext);

  const generateAddPatchData = useCallback(
    (formData: IDigicertFormData) => {
      if (!config) return null;

      const data: { integrations: Partial<IGlobalIntegrations> } = {
        integrations: {},
      };

      switch (certAuthorityType) {
        case "ndes":
          break;
        case "digicert":
          data.integrations.digicert = [
            ...(config.integrations.digicert || []),
            {
              name: formData.name,
              url: formData.url,
              api_token: formData.apiToken,
              profile_id: formData.profileId,
              certificate_common_name: formData.commonName,
              certificate_user_principal_names: [formData.userPrincipalName],
              certificate_seat_id: formData.certificateSeatId,
            },
          ];
          break;
        case "custom":
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
              (certAuthority as ICertificatesIntegrationCustomSCEP).id ===
              cert.id
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
    (formData: IDigicertFormData) => {
      if (!config) return null;

      const data: { integrations: Partial<IGlobalIntegrations> } = {
        integrations: {},
      };

      switch (certAuthorityType) {
        case "ndes":
          break;
        case "digicert":
          data.integrations.digicert = config.integrations.digicert?.map(
            (cert) => {
              // only update the certificate authority that we are editing
              if (
                (certAuthority as ICertificatesIntegrationDigicert).name ===
                cert.name
              ) {
                return {
                  name: formData.name,
                  url: formData.url,
                  api_token: formData.apiToken,
                  profile_id: formData.profileId,
                  certificate_common_name: formData.commonName,
                  certificate_user_principal_names: [
                    formData.userPrincipalName,
                  ],
                  certificate_seat_id: formData.certificateSeatId,
                };
              }
              return cert;
            }
          );
          break;
        case "custom":
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
    default:
      return "";
  }
};

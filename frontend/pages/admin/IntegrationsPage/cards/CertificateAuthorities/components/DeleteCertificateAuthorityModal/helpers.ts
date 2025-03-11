import { AppContext } from "context/app";

import {
  ICertificateAuthorityType,
  ICertificateIntegration,
  ICertificatesIntegrationCustomSCEP,
  ICertificatesIntegrationDigicert,
  IGlobalIntegrations,
} from "interfaces/integration";
import { useContext } from "react";

export const useCertAuthorityDataGenerator = (
  certAuthorityType: ICertificateAuthorityType,
  certAuthority: ICertificateIntegration
) => {
  const { config } = useContext(AppContext);

  /**
   * generates the data to be sent to the API to delete the certificate authority.
   * under the hood we are updating the app config object with the new data and
   * have to generate the correct data for the PATCH request.
   */
  const generateDeletePatchData = () => {
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
              (certAuthority as ICertificatesIntegrationDigicert).id === cert.id
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
  };

  /**
   * generates the data to be sent to the API to edit the certificate authority.
   * under the hood we are updating the app config object with the new data and
   * have to generate the correct data for the PATCH request.
   */
  const generateEditPatchData = () => {
    if (!config) return null;
  };

  return {
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

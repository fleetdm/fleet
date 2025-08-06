import { IAddCertAuthorityBody } from "services/entities/certificates";

import { ICertificateAuthorityType } from "interfaces/certificates";
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
// eslint-disable-next-line import/prefer-default-export
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
      const {
        name: hydrantName,
        url,
        clientId,
        clientSecret,
      } = formData as IHydrantFormData;
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
  }
};

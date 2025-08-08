import endpoints from "utilities/endpoints";
import sendRequest from "services";
import {
  ICertificateAuthorityPartial,
  ICertificateAuthority,
  ICertificatesCustomSCEP,
  ICertificatesDigicert,
  ICertificatesHydrant,
  ICertificatesNDES,
} from "interfaces/certificates";

type IGetCertAuthoritiesListResponse = ICertificateAuthorityPartial[];

type IAddCertAuthorityResponse = ICertificateAuthorityPartial;

type IGetCertAuthorityResponse = ICertificateAuthority;

interface IRequestCertAuthorityResponse {
  certificate: string;
}

export type IAddCertAuthorityBody =
  | { digicert: ICertificatesDigicert }
  | { ndes_scep_proxy: ICertificatesNDES }
  | { custom_scep_proxy: ICertificatesCustomSCEP }
  | { hydrant: ICertificatesHydrant };

export type IEditCertAuthorityBody =
  | { digicert: Partial<ICertificatesDigicert> }
  | { ndes_scep_proxy: Partial<ICertificatesNDES> }
  | { custom_scep_proxy: Partial<ICertificatesCustomSCEP> }
  | { hydrant: Partial<ICertificatesHydrant> };

export default {
  getCertificateAuthoritiesList: (): Promise<IGetCertAuthoritiesListResponse> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    // return sendRequest("GET", CERTIFICATE_AUTHORITIES);
    return new Promise((resolve) => {
      resolve([
        {
          id: 1,
          name: "DigiCert CA",
          type: "digicert",
        },
        {
          id: 2,
          name: "Example CA",
          type: "ndes_scep_proxy",
        },
        { id: 3, name: "Custom SCEP CA", type: "custom_scep_proxy" },
        { id: 4, name: "Hydrant CA", type: "hydrant" },
      ]);
    });
  },

  getCertificateAuthority: (id: number): Promise<IGetCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    // return sendRequest("GET", CERTIFICATE_AUTHORITY(id));
    return new Promise((resolve) => {
      resolve({
        id: 1,
        name: "Example_CA",
        type: "digicert",
        url: "https://example.com",
        api_token: "********",
        profile_id: "profile123",
        certificate_common_name: "example.com",
        certificate_user_principal_names: ["test@example.com"],
        certificate_seat_id: "seat123",
      } as ICertificatesDigicert);
    });
  },

  addCertificateAuthority: (
    certData: IAddCertAuthorityBody
  ): Promise<IAddCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("POST", CERTIFICATE_AUTHORITIES, certData);
  },

  editCertificateAuthority: (
    id: number,
    updateData: IEditCertAuthorityBody
  ): Promise<void> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("PATCH", CERTIFICATE_AUTHORITY(id), updateData);
  },

  deleteCertificateAuthority: (id: number): Promise<void> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("DELETE", CERTIFICATE_AUTHORITY(id));
  },

  requestCertificate: (id: number): Promise<IRequestCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITY_REQUEST_CERT } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY_REQUEST_CERT(id));
  },
};

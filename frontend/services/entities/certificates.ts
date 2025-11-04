import endpoints from "utilities/endpoints";
import sendRequest from "services";
import {
  ICertificateAuthorityPartial,
  ICertificateAuthority,
  ICertificatesCustomSCEP,
  ICertificatesDigicert,
  ICertificatesHydrant,
  ICertificatesNDES,
  ICertificatesSmallstep,
  ICertificatesCustomEST,
} from "interfaces/certificates";

type IGetCertAuthoritiesListResponse = {
  certificate_authorities: ICertificateAuthorityPartial[];
};

type IAddCertAuthorityResponse = ICertificateAuthorityPartial;

type IGetCertAuthorityResponse = ICertificateAuthority;

interface IRequestCertAuthorityResponse {
  certificate: string;
}

export type IAddCertAuthorityBody =
  | { digicert: ICertificatesDigicert }
  | { ndes_scep_proxy: ICertificatesNDES }
  | { custom_scep_proxy: ICertificatesCustomSCEP }
  | { hydrant: ICertificatesHydrant }
  | { smallstep: ICertificatesSmallstep }
  | { custom_est_proxy: ICertificatesCustomEST };

export type IEditCertAuthorityBody =
  | { digicert: Partial<ICertificatesDigicert> }
  | { ndes_scep_proxy: Partial<ICertificatesNDES> }
  | { custom_scep_proxy: Partial<ICertificatesCustomSCEP> }
  | { hydrant: Partial<ICertificatesHydrant> }
  | { smallstep: Partial<ICertificatesSmallstep> }
  | { custom_est_proxy: Partial<ICertificatesCustomEST> };

export default {
  getCertificateAuthoritiesList: (): Promise<IGetCertAuthoritiesListResponse> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITIES);
  },

  getCertificateAuthority: (id: number): Promise<IGetCertAuthorityResponse> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY(id));
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

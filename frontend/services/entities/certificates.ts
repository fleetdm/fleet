import endpoints from "utilities/endpoints";
import sendRequest from "services";
import {
  ICertificateAuthority,
  ICertificateAuthorityType,
  ICertificatesCustomSCEP,
  ICertificatesDigicert,
  ICertificatesHydrant,
  ICertificatesNDES,
} from "interfaces/certificates";

/** This interface represent the smaller subset of data that is returned for
some of the cert authority endpoints */
interface ICertAuthPartialResponse {
  id: number;
  name: string;
  type: ICertificateAuthorityType;
}

type IGetCertAuthoritiesListResponse = ICertAuthPartialResponse[];

type IAddCertAuthorityResponse = ICertAuthPartialResponse;

type IGetCertAuthorityResponse = ICertificateAuthority;

interface IRequestCertAuthorityResponse {
  certificate: string;
}

type IAddCertAuthorityBody =
  | { digicert: ICertificatesDigicert }
  | { ndes_scep_proxy: ICertificatesNDES }
  | { custom_scep_proxy: ICertificatesCustomSCEP }
  | { hydrant: ICertificatesHydrant };

type IEditCertAuthorityBody =
  | { digicert: Partial<ICertificatesDigicert> }
  | { ndes_scep_proxy: Partial<ICertificatesNDES> }
  | { custom_scep_proxy: Partial<ICertificatesCustomSCEP> }
  | { hydrant: Partial<ICertificatesHydrant> };

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

  editCertAuthorityModal: (
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

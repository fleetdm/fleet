import endpoints from "utilities/endpoints";
import sendRequest from "services";

export default {
  getCertificateAuthoritiesList: (): Promise<any> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITIES);
  },

  getCertificateAuthority: (id: number): Promise<any> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY(id));
  },

  addCertificateAuthority: (updateData: any): Promise<any> => {
    const { CERTIFICATE_AUTHORITIES } = endpoints;
    return sendRequest("POST", CERTIFICATE_AUTHORITIES, updateData);
  },

  editCertAuthorityModal: (id: number, updateData: any): Promise<any> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("PATCH", CERTIFICATE_AUTHORITY(id), updateData);
  },

  deleteCertificateAuthority: (id: number): Promise<any> => {
    const { CERTIFICATE_AUTHORITY } = endpoints;
    return sendRequest("DELETE", CERTIFICATE_AUTHORITY(id));
  },

  requestCertificate: (id: number): Promise<any> => {
    const { CERTIFICATE_AUTHORITY_REQUEST_CERT } = endpoints;
    return sendRequest("GET", CERTIFICATE_AUTHORITY_REQUEST_CERT(id));
  },
};

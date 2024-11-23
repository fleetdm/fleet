import { IPkiCert, IPkiConfig, IPkiTemplate } from "interfaces/pki";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

export interface IPkiListCertsResponse {
  certificates: IPkiCert[];
}

const pkiServive = {
  listCerts: (): Promise<IPkiListCertsResponse> => {
    return sendRequest("GET", endpoints.PKI);
  },

  getCert: (pkiName: string) => {
    const path = `${endpoints.PKI}/${pkiName}`;
    return sendRequest("GET", path);
  },

  uploadCert: (pkiName: string, certFile: File) => {
    const path = `${endpoints.PKI}/${pkiName}`;
    const formData = new FormData();
    formData.append("certificate", certFile);

    return sendRequest("POST", path, formData);
  },

  // TODO: when cert is deleted should the backend also update app config to delete the associated integrations/templates
  deleteCert: (pkiName: string) => {
    const path = `${endpoints.PKI}/${pkiName}`;
    return sendRequest("DELETE", path);
  },

  requestCSR: (pkiName: string) => {
    const path = `${endpoints.PKI}/${pkiName}/request_csr`;
    return sendRequest("GET", path);
  },

  addTemplate: (pkiName: string, template: IPkiTemplate) => {
    const { CONFIG } = endpoints;
    const formData = {
      integrations: {
        digicert_pki: [
          {
            pki_name: pkiName,
            templates: [template],
          },
        ],
      },
    };

    return sendRequest("PATCH", CONFIG, formData, undefined, undefined);
  },
};

export default pkiServive;

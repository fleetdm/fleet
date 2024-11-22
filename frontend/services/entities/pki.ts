import { IPkiTemplate } from "interfaces/pki";
import sendRequest from "services";
import endpoints from "utilities/endpoints";

const pkiServive = {
  listCerts: () => {
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
            name: pkiName,
            templates: [template],
          },
        ],
      },
    };

    return sendRequest("PATCH", CONFIG, formData, undefined, undefined);
  },
};

export default pkiServive;

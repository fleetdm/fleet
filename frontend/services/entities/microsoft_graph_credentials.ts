import sendRequest from "services";
import endpoints from "utilities/endpoints";
import {
  IMicrosoftGraphCredential,
  IMicrosoftGraphCredentialFormData,
} from "interfaces/microsoft_graph_credential";

export interface IGetMicrosoftGraphCredentialsResponse {
  microsoft_graph_credentials: IMicrosoftGraphCredential[];
}

export default {
  getCredentials: (): Promise<IGetMicrosoftGraphCredentialsResponse> => {
    const { MDM_MICROSOFT_GRAPH_CREDENTIALS } = endpoints;
    return sendRequest("GET", MDM_MICROSOFT_GRAPH_CREDENTIALS);
  },

  applyCredentials: (
    credentials: IMicrosoftGraphCredentialFormData[]
  ): Promise<void> => {
    const { MDM_MICROSOFT_GRAPH_CREDENTIALS } = endpoints;
    return sendRequest("PUT", MDM_MICROSOFT_GRAPH_CREDENTIALS, {
      microsoft_graph_credentials: credentials,
    });
  },

  deleteCredentials: (): Promise<void> => {
    const { MDM_MICROSOFT_GRAPH_CREDENTIALS } = endpoints;
    return sendRequest("PUT", MDM_MICROSOFT_GRAPH_CREDENTIALS, {
      microsoft_graph_credentials: [],
    });
  },
};

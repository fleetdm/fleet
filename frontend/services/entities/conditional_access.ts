import sendRequest from "services";
import endpoints from "utilities/endpoints";

export type TriggerMSConditionalStatusResponse = {
  microsoft_authentication_url: string;
};
export type ConfirmMSConditionalAccessResponse = {
  configuration_completed: boolean;
  setup_error: string;
};
export interface IConditionalAccessWindowsProfilesResponse {
  scep_profile: string;
  browser_profile: string;
}

const conditionalAccessService = {
  triggerMicrosoftConditionalAccess: (
    msTenantId: string
  ): Promise<TriggerMSConditionalStatusResponse> => {
    return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_MICROSOFT, {
      microsoft_tenant_id: msTenantId,
    });
  },
  confirmMicrosoftConditionalAccess: (): Promise<ConfirmMSConditionalAccessResponse> => {
    return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_MICROSOFT_CONFIRM);
  },
  deleteMicrosoftConditionalAccess: () => {
    return sendRequest("DELETE", endpoints.CONDITIONAL_ACCESS_MICROSOFT);
  },
  getIdpAppleProfile: (): Promise<string> => {
    return sendRequest(
      "GET",
      endpoints.CONDITIONAL_ACCESS_IDP_APPLE_PROFILE,
      undefined,
      "text"
    );
  },
  getIdpWindowsProfiles: (): Promise<IConditionalAccessWindowsProfilesResponse> => {
    return sendRequest(
      "GET",
      endpoints.CONDITIONAL_ACCESS_IDP_WINDOWS_PROFILES
    );
  },
  getIdpSigningCert: (): Promise<Blob> => {
    return sendRequest(
      "GET",
      endpoints.CONDITIONAL_ACCESS_IDP_SIGNING_CERT,
      undefined,
      "blob"
    );
  },
};

export default conditionalAccessService;

import sendRequest from "services";

import endpoints from "utilities/endpoints";

export type TriggerMSConditionalStatusResponse = {
  microsoft_authentication_url: string;
};
export type ConfirmMSConditionalAccessResponse = {
  configuration_completed: boolean;
};

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
};

export default conditionalAccessService;

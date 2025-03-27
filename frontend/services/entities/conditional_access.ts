import sendRequest from "services";

import endpoints from "utilities/endpoints";

export type TriggerMSConditionalStatusResponse = {
  microsoft_authentication_url: string;
};
export type ConfirmMSConditionalStatusResponse = { admin_consented: boolean };

const conditionalAccessService = {
  triggerMicrosoftConditionalAccess: (msTenantId: string) => {
    return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_MICROSOFT, {
      microsoft_tenant_id: msTenantId,
    });
  },
  confirmMicrosoftConditionalAccess: () => {
    return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_CONFIRM);
  },
  deleteMicrosoftConditionalAccess: () => {
    return sendRequest("DELETE", endpoints.CONDITIONAL_ACCESS_MICROSOFT);
  },
};

export default conditionalAccessService;

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
    // TODO - real ones!

    // return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_MICROSOFT, {
    //   microsoft_tenant_id: msTenantId,
    // });
    return Promise.resolve({
      microsoft_authentication_url: "https://www.example.com",
    });
  },
  confirmMicrosoftConditionalAccess: (): Promise<ConfirmMSConditionalAccessResponse> => {
    // TODO - real one!
    // return sendRequest("POST", endpoints.CONDITIONAL_ACCESS_MICROSOFT_CONFIRM);

    return Promise.resolve({
      configuration_completed: true,
      // configuration_completed: false,
    });

    // return Promise.reject(new Error("Bad data"));
  },
  deleteMicrosoftConditionalAccess: () => {
    // TODO - real one!
    // return sendRequest("DELETE", endpoints.CONDITIONAL_ACCESS_MICROSOFT);
    return Promise.resolve();
  },
};

export default conditionalAccessService;

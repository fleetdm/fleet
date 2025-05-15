import sendRequest from "services";
import endpoints from "utilities/endpoints";
import { authToken } from "utilities/local";

interface IGetAndroidSignupUrlResponse {
  android_enterprise_signup_url: string;
}

interface IGetAndroidEnterpriseResponse {
  android_enterprise_id: boolean;
}

export default {
  getSignupUrl: (): Promise<IGetAndroidSignupUrlResponse> => {
    const { MDM_ANDROID_SIGNUP_URL } = endpoints;
    return sendRequest("GET", MDM_ANDROID_SIGNUP_URL);
  },

  getAndroidEnterprise: (): Promise<IGetAndroidEnterpriseResponse> => {
    const { MDM_ANDROID_ENTERPRISE } = endpoints;
    return sendRequest("GET", MDM_ANDROID_ENTERPRISE);
  },

  turnOffAndroidMdm: (): Promise<void> => {
    const { MDM_ANDROID_ENTERPRISE } = endpoints;
    return sendRequest("DELETE", MDM_ANDROID_ENTERPRISE);
  },

  /**
   * This function starts a Server-Sent Events connection with the fleet server
   * to get messages about a successful Android mdm connection. We have to use
   * fetch here because the EventSource API does not support setting headers,
   * which we need to authenticate the request.
   */
  startSSE: (abortSignal: AbortSignal): Promise<void> => {
    return new Promise(async (resolve, reject) => {
      try {
        const response = await fetch(endpoints.MDM_ANDROID_SSE_URL, {
          method: "GET",
          headers: {
            Authorization: `Bearer ${authToken()}`,
          },
          signal: abortSignal,
        });

        const reader = response?.body?.getReader();
        const decoder = new TextDecoder();

        while (true) {
          // @ts-ignore
          // eslint-disable-next-line no-await-in-loop
          const { done, value } = await reader?.read();
          if (done) break;
          const text = decoder.decode(value);
          if (text === "Android Enterprise successfully connected") {
            resolve();
            break;
          }
        }
      } catch (error) {
        if ((error as Error).name === "AbortError") {
          // we want to ignore abort errors
          console.error("SSE Fetch aborted");
        } else {
          reject(error);
        }
      }
    });
  },
};

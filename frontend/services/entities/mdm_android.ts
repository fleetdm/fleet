import sendRequest from "services";
import endpoints from "utilities/endpoints";
import authToken from "utilities/auth_token";

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
            Authorization: `Bearer ${authToken.get()}`,
          },
          signal: abortSignal,
        });

        const reader = response?.body?.getReader();
        if (!reader) {
          reject(new Error("Android MDM SSE stream unavailable"));
          return;
        }
        const decoder = new TextDecoder();
        const successSignal = "Android Enterprise successfully connected";
        // Buffer accumulates decoded text so a success message split across
        // multiple chunks (valid with chunked transfer encoding) is still
        // detected.
        let buffer = "";

        while (true) {
          // eslint-disable-next-line no-await-in-loop
          const { done, value } = await reader.read();
          if (done) {
            // Server closed the stream without ever sending success.
            // Reject so callers don't await forever on unmount or backend hiccup.
            reject(new Error("Android MDM SSE ended before success signal"));
            return;
          }
          buffer += decoder.decode(value, { stream: true });
          if (buffer.includes(successSignal)) {
            resolve();
            return;
          }
          buffer = buffer.slice(-successSignal.length);
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

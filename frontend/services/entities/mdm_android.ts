import sendRequest from "services";
import endpoints from "utilities/endpoints";

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
};

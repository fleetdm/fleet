import sendRequest from "services";
import endpoints from "utilities/endpoints";

interface IGetAndroidSignupUrlResponse {
  android_enterprise_signup_url: string;
}

export default {
  getSignupUrl: (): Promise<IGetAndroidSignupUrlResponse> => {
    const { MDM_ANDROID_SIGNUP_URL } = endpoints;
    return sendRequest("GET", MDM_ANDROID_SIGNUP_URL);
  },
};

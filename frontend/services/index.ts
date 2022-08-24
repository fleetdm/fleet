import axios, {
  AxiosError,
  AxiosResponse,
  ResponseType as AxiosResponseType,
} from "axios";
import local from "utilities/local";
import URL_PREFIX from "router/url_prefix";

const sendRequest = async (
  method: "GET" | "POST" | "PATCH" | "DELETE" | "HEAD",
  path: string,
  data?: unknown,
  responseType: AxiosResponseType = "json"
): Promise<any> => {
  const { origin } = global.window.location;

  const url = `${origin}${URL_PREFIX}/api${path}`;
  const token = local.getItem("auth_token");

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    return Promise.resolve(response.data);
  } catch (error) {
    const axiosError = error as AxiosError;
    return Promise.reject(axiosError.response);
  }
};

// return the first error
export const getError = (response: unknown): string => {
  const r = response as AxiosResponse;
  return r.data?.errors?.[0]?.reason || ""; // TODO: check if any callers rely on empty return value
};

export default sendRequest;

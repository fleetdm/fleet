import axios, {
  AxiosError,
  AxiosResponse,
  ResponseType as AxiosResponseType,
} from "axios";
import { authToken } from "utilities/local";

import URL_PREFIX from "router/url_prefix";

const sendRequest = async (
  method: "GET" | "POST" | "PATCH" | "DELETE" | "HEAD",
  path: string,
  data?: unknown,
  responseType: AxiosResponseType = "json",
  timeout?: number,
  includeFullAxiosError?: boolean
) => {
  const { origin } = global.window.location;

  const url = `${origin}${URL_PREFIX}/api${path}`;
  const token = authToken();

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      timeout,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    return response.data;
  } catch (error) {
    if (includeFullAxiosError) {
      return Promise.reject(error);
    }
    const axiosError = error as AxiosError;
    return Promise.reject(
      axiosError.response ||
        axiosError.message ||
        axiosError.code ||
        "unknown axios error"
    );
  }
};

// return the first error
export const getError = (response: unknown): string => {
  const r = response as AxiosResponse;
  return r.data?.errors?.[0]?.reason || ""; // TODO: check if any callers rely on empty return value
};

export default sendRequest;

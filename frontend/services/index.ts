import axios, { ResponseType as AxiosResponseType } from "axios";
import URL_PREFIX from "router/url_prefix";
import { authToken } from "utilities/local";
import { parseAxiosError } from "./errors";

export const sendRequest = async (
  method: "GET" | "POST" | "PATCH" | "DELETE" | "HEAD",
  path: string,
  data?: unknown,
  responseType: AxiosResponseType = "json",
  timeout?: number,
  skipParseError?: boolean
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
    if (skipParseError) {
      return Promise.reject(error);
    }
    const axiosError = parseAxiosError(error);
    return Promise.reject(
      axiosError?.response ||
        axiosError?.message ||
        axiosError?.code ||
        `send request: parse server error: ${error}`
    );
  }
};

export default sendRequest;

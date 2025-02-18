import axios, {
  isAxiosError,
  ResponseType as AxiosResponseType,
  AxiosProgressEvent,
} from "axios";
import URL_PREFIX from "router/url_prefix";
import { authToken } from "utilities/local";

export const sendRequestWithProgress = async ({
  method,
  path,
  data,
  responseType = "json",
  timeout,
  skipParseError,
  returnRaw,
  onDownloadProgress,
  onUploadProgress,
  signal,
}: {
  method: "GET" | "POST" | "PATCH" | "DELETE" | "HEAD";
  path: string;
  data?: unknown;
  responseType?: AxiosResponseType;
  timeout?: number;
  skipParseError?: boolean;
  returnRaw?: boolean;
  onDownloadProgress?: (progressEvent: AxiosProgressEvent) => void;
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void;
  signal?: AbortSignal;
}) => {
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
      onDownloadProgress,
      onUploadProgress,
      signal,
    });

    if (returnRaw) {
      return response;
    }
    return response.data;
  } catch (error) {
    if (skipParseError) {
      return Promise.reject(error);
    }
    let reason: unknown | undefined;
    if (isAxiosError(error)) {
      reason = error.response || error.message || error.code;
    }
    return Promise.reject(
      reason || `send request: parse server error: ${error}`
    );
  }
};

export const sendRequest = async (
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD",
  path: string,
  data?: unknown,
  responseType: AxiosResponseType = "json",
  timeout?: number,
  skipParseError?: boolean,
  returnRaw?: boolean
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

    if (returnRaw) {
      return response;
    }
    return response.data;
  } catch (error) {
    if (skipParseError) {
      return Promise.reject(error);
    }
    let reason: unknown | undefined;
    if (isAxiosError(error)) {
      reason = error.response || error.message || error.code;
    }
    return Promise.reject(
      reason || `send request: parse server error: ${error}`
    );
  }
};

export default sendRequest;

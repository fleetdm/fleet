import axios, {
  isAxiosError,
  ResponseType as AxiosResponseType,
  AxiosProgressEvent,
} from "axios";

import URL_PREFIX from "router/url_prefix";
import authToken from "utilities/auth_token";
import endpoints from "utilities/endpoints";

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

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      timeout,
      headers: {
        Authorization: `Bearer ${authToken.get()}`,
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

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      timeout,
      headers: {
        Authorization: `Bearer ${authToken.get()}`,
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

/**
 * Send a request with custom headers. Used for requests that need to signal
 * special handling (e.g., base64-encoded scripts to bypass WAF rules).
 */
export const sendRequestWithHeaders = async (
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "HEAD",
  path: string,
  data?: unknown,
  customHeaders?: Record<string, string>,
  responseType: AxiosResponseType = "json",
  timeout?: number,
  skipParseError?: boolean,
  returnRaw?: boolean
) => {
  const { origin } = global.window.location;

  const url = `${origin}${URL_PREFIX}/api${path}`;

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      timeout,
      headers: {
        Authorization: `Bearer ${authToken.get()}`,
        ...customHeaders,
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

/**
 * Send a request with progress tracking and custom headers. Used for file uploads
 * that need to signal special handling (e.g., base64-encoded scripts to bypass WAF rules).
 */
export const sendRequestWithProgressAndHeaders = async ({
  method,
  path,
  data,
  customHeaders,
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
  customHeaders?: Record<string, string>;
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

  try {
    const response = await axios({
      method,
      url,
      data,
      responseType,
      timeout,
      headers: {
        Authorization: `Bearer ${authToken.get()}`,
        ...customHeaders,
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

/**
 * Uploads a file straight to object storage through a presigned URL and returns
 * the upload id to register it with. The PUT uses bare axios so Fleet
 * credentials never reach the storage host; don't route it through the helpers
 * above.
 */
export const uploadToStorage = async ({
  target,
  file,
  teamId,
  onUploadProgress,
  signal,
}: {
  target: "software_package" | "bootstrap_package";
  file: File;
  teamId?: number;
  onUploadProgress?: (progressEvent: AxiosProgressEvent) => void;
  signal?: AbortSignal;
}): Promise<string> => {
  const { upload_id, url } = await sendRequest(
    "POST",
    endpoints.STAGED_UPLOAD,
    {
      target,
      fleet_id: teamId ?? 0,
      size: file.size,
    }
  );
  await axios.put(url, file, { onUploadProgress, signal });
  return upload_id;
};

export default sendRequest;

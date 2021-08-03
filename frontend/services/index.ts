import axios from "axios";
// @ts-ignore
import local from "utilities/local";
import URL_PREFIX from "router/url_prefix";

const createErrorMessage = (error: any): string => {
  if (error.response) {
    return error.response.data;
  } else if (error.request) {
    return "A connection error occurred. Please try again or contact us.";
  }

  return "Something went wrong. Please try again or contact us.";
};

const sendRequest = async (
  method: "GET" | "POST" | "PATCH" | "DELETE",
  path: string,
  data?: any
) => {
  const { origin } = global.window.location;

  const url = `${origin}${URL_PREFIX}/api${path}`;
  const token = local.getItem("auth_token");

  try {
    const response = await axios({
      method,
      url,
      data,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    return Promise.resolve(response.data);
  } catch (error) {
    const message = createErrorMessage(error);
    return Promise.reject(message);
  }
};

export default sendRequest;

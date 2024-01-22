import { get, join } from "lodash";
import { IFleetApiError } from "interfaces/errors";

const formatServerErrors = (errors: IFleetApiError[]) => {
  if (!errors || !errors.length) {
    return {};
  }

  const result: { [key: string]: string } = {};

  errors.forEach((error) => {
    const { name, reason } = error;

    if (result[name]) {
      result[name] = join([result[name], reason], ", ");
    } else {
      result.base = reason; // Ensure a base error is always returned
    }
  });

  return result; // TODO: Typing {base: string}
};

const formatErrorResponse = (errorResponse: any) => {
  const errors =
    get(errorResponse, "message.errors") ||
    get(errorResponse, "data.errors") ||
    [];

  return {
    ...formatServerErrors(errors),
    http_status: errorResponse.status,
  } as any; // TODO: Fix type to IOldApiError
};

export default formatErrorResponse;

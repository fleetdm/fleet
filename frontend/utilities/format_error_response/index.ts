import { get, join } from "lodash";

import { IFleetApiError } from "interfaces/errors";

const formatServerErrors = (errors: IFleetApiError[]) => {
  if (!errors || !errors.length) {
    return {};
  }

  const result: { [key: string]: string } = {};

  errors.forEach((error) => {
    const { name, reason } = error;

    // `base` is the global error (fed to toasts and callers that only read
    // `.base`); field-keyed entries let forms render the same message inline on
    // the offending field.
    if (!result.base) {
      result.base = reason;
    }
    if (name && name !== "base") {
      result[name] = result[name] ? join([result[name], reason], ", ") : reason;
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

import { get, join } from "lodash";
import { IError } from "interfaces/errors";

const formatServerErrors = (errors: IError[]) => {
  if (!errors || !errors.length) {
    return {};
  }

  const result: { [key: string]: string } = {};

  errors.forEach((error) => {
    const { name, reason } = error;

    if (result[name]) {
      result[name] = join([result[name], reason], ", ");
    } else {
      result[name] = reason;
    }
  });

  return result;
};

const formatErrorResponse = (errorResponse: any) => {
  const errors =
    get(errorResponse, "message.errors") ||
    get(errorResponse, "data.errors") ||
    [];

  return {
    ...formatServerErrors(errors),
    http_status: errorResponse.status,
  } as any;
};

export default formatErrorResponse;

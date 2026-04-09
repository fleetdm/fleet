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

const formatErrorResponse = (errorResponse: unknown) => {
  const errors =
    get(errorResponse, "message.errors") ||
    get(errorResponse, "data.errors") ||
    [];

  return {
    ...formatServerErrors(errors as IFleetApiError[]),
    http_status: get(errorResponse, "status") as number | undefined,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } as any; // TODO: Fix type to IOldApiError
};

export default formatErrorResponse;

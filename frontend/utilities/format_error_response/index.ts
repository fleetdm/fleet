import { get, join } from "lodash";

import { IFleetApiError } from "interfaces/errors";

const formatServerErrors = (errors: IFleetApiError[]) => {
  if (!errors || !errors.length) {
    return {};
  }

  const result: { [key: string]: string } = {};

  errors.forEach((error) => {
    const { name, reason } = error;

    // `base` is the global error read by toast-only callers (LoginPage,
    // ForgotPasswordPage, ResetPasswordPage). Preserve the legacy behavior:
    // repeated explicit `name:"base"` entries get joined with a comma; anything
    // else overwrites so the latest reason wins.
    if (name === "base" && result.base) {
      result.base = join([result.base, reason], ", ");
    } else {
      result.base = reason;
    }

    // Field-keyed entries are the new surface for forms that render inline
    // errors next to the offending field. Repeated field errors join with a
    // comma; `name` of `""` or `"base"` has no field to attach to and is
    // covered by `base` above.
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

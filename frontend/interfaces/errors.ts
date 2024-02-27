import PropTypes from "prop-types";
import { AxiosError, isAxiosError } from "axios";

export default PropTypes.shape({
  http_status: PropTypes.number,
  base: PropTypes.string,
});

// Response created by utilities/format_error_response
export interface IOldApiError {
  http_status: number;
  base: string;
}

/**
 * IFleetApiError is the shape of a Fleet API error. It represents an element of the `errors`
 * array in a Fleet API response for failed requests (see `IFleetApiResponseWithErrors`).
 */
export interface IFleetApiError {
  name: string;
  reason: string;
}

/**
 * IApiError is the shape of a Fleet API response for failed requests.
 *
 * TODO: Rename to IFleetApiResponseWithErrors
 */
export interface IApiError {
  message: string;
  errors: IFleetApiError[];
  uuid?: string;
}

const isFleetApiError = (err: unknown): err is IFleetApiError => {
  if (!err || typeof err !== "object" || !("name" in err && "reason" in err)) {
    return false;
  }
  const e = err as Record<"name" | "reason", unknown>;
  if (typeof e.name !== "string" || typeof e.reason !== "string") {
    return false;
  }
  return true;
};

interface IRecordWithErrors extends Record<string | number | symbol, unknown> {
  errors: unknown[];
}

const isRecordWithErrors = (r: unknown): r is IRecordWithErrors => {
  if (!r || typeof r !== "object" || !("errors" in r)) {
    return false;
  }
  const { errors } = r as { errors: unknown };
  if (!Array.isArray(errors)) {
    return false;
  }
  return true;
};

interface IRecordWithDataErrors
  extends Record<string | number | symbol, unknown> {
  data: IRecordWithErrors;
}

const isRecordWithDataErrors = (r: unknown): r is IRecordWithDataErrors => {
  if (!r || typeof r !== "object" || !("data" in r)) {
    return false;
  }
  const { data } = r as { data: unknown };
  if (!isRecordWithErrors(data)) {
    return false;
  }
  const { errors } = data;
  if (!Array.isArray(errors)) {
    return false;
  }
  return true;
};

interface IRecordWithResponseDataErrors
  extends Record<string | number | symbol, unknown> {
  response: IRecordWithDataErrors;
}

const isRecordWithResponseDataErrors = (
  r: unknown
): r is IRecordWithResponseDataErrors => {
  if (!r || typeof r !== "object" || !("response" in r)) {
    return false;
  }
  const { response } = r as { response: unknown };
  if (!isRecordWithDataErrors(response)) {
    return false;
  }
  return true;
};

interface IFilterFleetErrorBase {
  nameEquals?: string;
  reasonIncludes?: string;
}

interface IFilterFleetErrorName extends IFilterFleetErrorBase {
  nameEquals: string;
  reasonIncludes?: never;
}

interface IFilterFleetErrorReason extends IFilterFleetErrorBase {
  nameEquals?: never;
  reasonIncludes: string;
}

// FilterFleetError is the shape of a filter that can be applied to to filter Fleet
// server errors. It is the union of FilterFleetErrorName and FilterFleetErrorReason,
// which ensures that only one of `nameEquals` or `reasonIncludes` can be specified.
type IFilterFleetError = IFilterFleetErrorName | IFilterFleetErrorReason;

const filterFleetErrorNameEquals = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => isFleetApiError(e) && e.name === value) as
    | IFleetApiError
    | undefined;
};

const filterFleetErrorReasonIncludes = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => isFleetApiError(e) && e.reason?.includes(value)) as
    | IFleetApiError
    | undefined;
};

const getReasonFromErrors = (errors: unknown[], filter?: IFilterFleetError) => {
  if (!errors.length) {
    return "";
  }

  let fleetError: IFleetApiError | undefined;
  if (filter?.nameEquals) {
    fleetError = filterFleetErrorNameEquals(errors, filter.nameEquals);
  } else if (filter?.reasonIncludes) {
    fleetError = filterFleetErrorReasonIncludes(errors, filter.reasonIncludes);
  } else {
    fleetError = isFleetApiError(errors[0]) ? errors[0] : undefined;
  }
  return fleetError?.reason || "";
};

const getReasonFromRecordWithDataErrors = (
  r: IRecordWithDataErrors,
  filter?: IFilterFleetError
): string => {
  return getReasonFromErrors(r.data.errors, filter);
};

const getReasonFromAxiosError = (
  ae: AxiosError,
  filter?: IFilterFleetError
): string => {
  return isRecordWithDataErrors(ae.response)
    ? getReasonFromRecordWithDataErrors(ae.response, filter)
    : "";
};

/**
 * getErrorReason attempts to parse a unknown payload as an `AxiosError` or
 * other `Record`-like object with the general shape as follows:
 * `{ response: { data: { errors: unknown[] } } }`
 *
 * It attempts to extract a `reason` from a Fleet API error (i.e. an object
 * with `name` and `reason` properties) in the `errors` array, if present.
 * Other in values in the payload are generally ignored.
 *
 * If `filter` is specified, it attempts to find an error that satisfies the filter
 * and returns the `reason`, if found. Otherwise, it returns the `reason`
 * of the first error, if any.
 *
 * By default, an empty string is returned as the reason if no error is found.
 */
export const getErrorReason = (
  payload: unknown | undefined,
  filter?: IFilterFleetError
): string => {
  if (isAxiosError(payload)) {
    return getReasonFromAxiosError(payload, filter);
  }

  if (isRecordWithResponseDataErrors(payload)) {
    return getReasonFromRecordWithDataErrors(payload.response, filter);
  }

  if (isRecordWithDataErrors(payload)) {
    return getReasonFromRecordWithDataErrors(payload, filter);
  }

  if (isRecordWithErrors(payload)) {
    return getReasonFromErrors(payload.errors, filter);
  }

  return "";
};

export const ignoreAxiosError = (err: Error, ignoreStatuses: number[]) => {
  if (!isAxiosError(err)) {
    return false;
  }
  return !!err.response && ignoreStatuses.includes(err.response.status);
};

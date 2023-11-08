import { AxiosError } from "axios";

/**
 * isAxiosError returns true if the value is an Error and has an `isAxiosError`
 * property. It is a type guard used when parsing errors from Fleet server responses.
 */
const isAxiosError = (err: unknown): err is AxiosError => {
  if (!(err instanceof Error && "isAxiosError" in err)) {
    return false;
  }
  const e = err as AxiosError;

  return e.isAxiosError === true;
};

/**
 * parseAxiosError attempts to parse an unknown value as an AxiosError.
 */
export const parseAxiosError = (raw: unknown) => {
  return isAxiosError(raw) ? raw : undefined;
};

/**
 * IFleetServerError is the shape of a Fleet server error.
 * It is used to parse out errors from Fleet server responses.
 */
interface IFleetServerError {
  name: string;
  reason: string;
}

/**
 * isFleetServerError returns true if the value is an object with `name` and `reason`
 * properties. It is a type guard used when parsing errors from Fleet server responses.
 */
const isFleetServerError = (err: unknown): err is IFleetServerError => {
  if (!err || typeof err !== "object" || !("name" in err && "reason" in err)) {
    return false;
  }
  const e = err as Record<"name" | "reason", unknown>;
  if (typeof e.name !== "string" || typeof e.reason !== "string") {
    return false;
  }
  return true;
};

/**
 * parseFleetError attempts to parse an unknown value as a Fleet server error,
 * which is an object with `name` and `reason` properties.
 * It is used to parse out errors from Fleet server responses.
 * If the value is not a Fleet server error, it returns undefined.
 */
export const parseFleetError = (
  err: unknown
): IFleetServerError | undefined => {
  return isFleetServerError(err) ? err : undefined;
};

/**
 * IFilterFleetErrorBase is the shape of a filter that can be applied to to filter Fleet
 * server errors. It represents all possible filters. This base type is further narrowed
 * by the IFilterFleetErrorName and IFilterFleetErrorReason types.
 */
interface IFilterFleetErrorBase {
  nameEquals?: string;
  reasonIncludes?: string;
}

/**
 * IFilterFleetErrorName is the shape of a filter that can be applied to to filter Fleet
 * server errors by name. It serves to narrow the IFilterFleetErrorBase type by requiring
 * that only `nameEquals` can be specified.
 */
interface IFilterFleetErrorName extends IFilterFleetErrorBase {
  nameEquals: string;
  reasonIncludes?: never;
}

/**
 * IFilterFleetErrorReason is the shape of a filter that can be applied to to filter Fleet
 * server errors by reason. It serves to narrow the IFilterFleetErrorBase type by requiring
 * that only `reasonIncludes` can be specified.
 */
interface IFilterFleetErrorReason extends IFilterFleetErrorBase {
  nameEquals?: never;
  reasonIncludes: string;
}

/**
 * FilterFleetError is the shape of a filter that can be applied to to filter Fleet
 * server errors. It is the union of FilterFleetErrorName and FilterFleetErrorReason,
 * which ensures that only one of `nameEquals` or `reasonIncludes` can be specified.
 */
type IFilterFleetError = IFilterFleetErrorName | IFilterFleetErrorReason;

/**
 * filterFleetErrorNameEquals returns the first of the provided array of elements
 * with unknown types, where such element is a Fleet server error (i.e. an object
 * with `name` and `reason` properties) and such error has a `name` that equals
 * the specified `value`. If no such error is found, it returns undefined.
 *
 * Any element that is not a Fleet server error is ignored.
 */
const filterFleetErrorNameEquals = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => parseFleetError(e)?.name === value) as
    | IFleetServerError
    | undefined;
};

/**
 * filterFleetErrorReasonIncludes returns the first of the provided array of elements
 * with unknown types, where such element is a Fleet server error (i.e. an object
 * with `name` and `reason` properties) and such error has a `reason` that includes
 * the specified `value`. If no such error is found, it returns undefined.
 *
 * Any element that is not a Fleet server error is ignored.
 */
const filterFleetErrorReasonIncludes = (errs: unknown[], value: string) => {
  if (!value || !errs?.length) {
    return undefined;
  }
  return errs?.find((e) => parseFleetError(e)?.reason?.includes(value)) as
    | IFleetServerError
    | undefined;
};

/**
 * hasResponseDataErrors returns true if the value is an object with a `data` property
 * that is an object with an`errors` property that is an array. It is a type guard
 * used when parsing errors from Fleet server responses.
 */
const hasResponseDataErrors = (
  response: unknown
): response is { data: { errors: unknown[] } } => {
  if (!response || typeof response !== "object" || !("data" in response)) {
    return false;
  }
  const { data } = response as { data: unknown };
  if (!data || typeof data !== "object" || !("errors" in data)) {
    return false;
  }
  const { errors } = data as { errors: unknown };
  if (!Array.isArray(errors)) {
    return false;
  }
  return true;
};

/**
 * parseDataErrorsFromResponse attempts to parse an unknown value as an object
 * with a `data` property that is an object with an`errors` property that is an
 * array. If the value is not such an object, it returns undefined. Otherwise,
 * it returns the `errors` as an array of unknown values.
 */
export const parseDataErrorsFromResponse = (raw: unknown | undefined) => {
  return hasResponseDataErrors(raw) ? raw.data.errors : undefined;
};

/**
 * getFleetErrorReason accepts an array of elements of unknown types and attempts to parse
 * each element as a Fleet server error (i.e. an object with `name` and `reason` properties).
 * Any element that is not a Fleet server error is ignored. If `filter` is specified, it
 * attempts to find an error that satisfies the filter and returns the `reason`, if found.
 * Otherwise, it returns the `reason` of the first error, if any. By default, an empty string
 * is returned as the reason if no error is found.
 */
const getReasonFromFleetErrors = (
  errors: unknown[] | undefined,
  filter?: IFilterFleetError
) => {
  if (!errors?.length) {
    return "";
  }

  let fleetError: IFleetServerError | undefined;
  if (filter?.nameEquals) {
    fleetError = filterFleetErrorNameEquals(errors, filter.nameEquals);
  } else if (filter?.reasonIncludes) {
    fleetError = filterFleetErrorReasonIncludes(errors, filter.reasonIncludes);
  } else {
    fleetError = parseFleetError(errors[0]);
  }
  return fleetError?.reason || "";
};

/**
 * getFleetErrorReasonFromResponse attempts to parse an unknown value as an object
 * with a `data` property that is an object with an `errors` property that is an array
 * of one or more Fleet server errors (i.e. an object with `name` and `reason`
 * properties). Any array element that is not a Fleet server error is ignored.
 *
 * If `filter` is specified, it attempts to find an error that satisfies the filter
 * and returns the `reason`, if found. Otherwise, it returns the `reason`
 * of the first error, if any. By default, an empty string is returned as the reason
 * if no error is found.
 */
export const getFleetErrorReasonFromResponse = (
  raw: unknown | undefined,
  filter?: IFilterFleetError
): string => {
  return getReasonFromFleetErrors(parseDataErrorsFromResponse(raw), filter);
};

/**
 * getFleetErrorReasonFromAxiosError attempts to parse an unknown value as an AxiosError
 * and extract an error reason from any Fleet server errors.
 *
 * If `filter` is specified, it attempts to find an error that satisfies the filter
 * and returns the `reason`, if found. Otherwise, it returns the `reason`
 * of the first error, if any. By default, an empty string is returned as the reason
 * if no error is found.
 */
export const getFleetErrorReasonFromAxiosError = (
  raw: unknown | undefined,
  filter?: IFilterFleetError
): string => {
  return getFleetErrorReasonFromResponse(
    parseAxiosError(raw)?.response,
    filter
  );
};

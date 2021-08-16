import { isString, isPlainObject, isEmpty, reduce, trim, union } from "lodash";

export const getNextLocationUrl = (
  options = {
    pathPrefix: "",
    newRouteTemplate: "",
    newRouteParams: {},
    newQueryParams: {},
  }
): string => {
  const pathPrefix = isString(options.pathPrefix) ? options.pathPrefix : "";
  const newRouteTemplate =
    (isString(options.newRouteTemplate) && options.newRouteTemplate) || "";
  const newRouteParams = isPlainObject(options.newRouteParams)
    ? options.newRouteParams
    : {};
  const newQueryParams = isPlainObject(options.newQueryParams)
    ? options.newQueryParams
    : {};

  let routeString = "";

  if (!isEmpty(newRouteParams)) {
    routeString = reduce(
      newRouteParams,
      (string, value, key) => {
        return string.replace(`:${key}`, encodeURIComponent(value));
      },
      newRouteTemplate
    );
  }

  let queryString = "";
  if (!isEmpty(newQueryParams)) {
    queryString = reduce(
      newQueryParams,
      (arr: string[], value, key) => {
        key && arr.push(`${key}=${encodeURIComponent(value)}`);
        return arr;
      },
      []
    ).join("&");
  }

  const nextLocation = union(
    trim(pathPrefix, "/").split("/"),
    routeString.split("/")
  ).join("/");

  return queryString ? `/${nextLocation}?${queryString}` : `/${nextLocation}`;
};

export default { getNextLocationUrl };

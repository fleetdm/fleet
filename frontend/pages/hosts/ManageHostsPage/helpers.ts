import { isString, isPlainObject, isEmpty, reduce, trim, union } from "lodash";

export const getNextLocationPath = (
  options = {
    pathPrefix: "",
    routeTemplate: "",
    routeParams: {},
    queryParams: {},
  }
): string => {
  const pathPrefix = isString(options.pathPrefix) ? options.pathPrefix : "";
  const routeTemplate =
    (isString(options.routeTemplate) && options.routeTemplate) || "";
  const routeParams = isPlainObject(options.routeParams)
    ? options.routeParams
    : {};
  const queryParams = isPlainObject(options.queryParams)
    ? options.queryParams
    : {};

  let routeString = "";

  if (!isEmpty(routeParams)) {
    routeString = reduce(
      routeParams,
      (string, value, key) => {
        return string.replace(`:${key}`, encodeURIComponent(value));
      },
      routeTemplate
    );
  }

  let queryString = "";
  if (!isEmpty(queryParams)) {
    queryString = reduce(
      queryParams,
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

export default { getNextLocationPath };

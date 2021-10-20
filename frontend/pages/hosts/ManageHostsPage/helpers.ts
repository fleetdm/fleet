import { isString, isPlainObject, isEmpty, reduce, trim, union } from "lodash";

interface ILocationParams {
  pathPrefix?: string;
  routeTemplate?: string;
  routeParams?: { [key: string]: any };
  queryParams?: { [key: string]: any };
}

export const NEW_LABEL_HASH = "#new_label";
export const EDIT_LABEL_HASH = "#edit_label";
export const ALL_HOSTS_LABEL = "all-hosts";
export const LABEL_SLUG_PREFIX = "labels/";

export const DEFAULT_SORT_HEADER = "hostname";
export const DEFAULT_SORT_DIRECTION = "asc";

export const HOST_SELECT_STATUSES = [
  {
    disabled: false,
    label: "All hosts",
    value: ALL_HOSTS_LABEL,
    helpText: "All hosts which have enrolled to Fleet.",
  },
  {
    disabled: false,
    label: "Online hosts",
    value: "online",
    helpText: "Hosts that have recently checked-in to Fleet.",
  },
  {
    disabled: false,
    label: "Offline hosts",
    value: "offline",
    helpText: "Hosts that have not checked-in to Fleet recently.",
  },
  {
    disabled: false,
    label: "New hosts",
    value: "new",
    helpText: "Hosts that have been enrolled to Fleet in the last 24 hours.",
  },
  {
    disabled: false,
    label: "MIA hosts",
    value: "mia",
    helpText: "Hosts that have not been seen by Fleet in more than 30 days.",
  },
];

export const isAcceptableStatus = (filter: string) => {
  return (
    filter === "new" ||
    filter === "online" ||
    filter === "offline" ||
    filter === "mia"
  );
};

export const isValidPolicyResponse = (filter: string) => {
  return filter === "pass" || filter === "fail";
};

export const getNextLocationPath = ({
  pathPrefix = "",
  routeTemplate = "",
  routeParams = {},
  queryParams = {},
}: ILocationParams): string => {
  const pathPrefixFinal = isString(pathPrefix) ? pathPrefix : "";
  const routeTemplateFinal = (isString(routeTemplate) && routeTemplate) || "";
  const routeParamsFinal = isPlainObject(routeParams) ? routeParams : {};
  const queryParamsFinal = isPlainObject(queryParams) ? queryParams : {};

  let routeString = "";

  if (!isEmpty(routeParamsFinal)) {
    routeString = reduce(
      routeParamsFinal,
      (string, value, key) => {
        return string.replace(`:${key}`, encodeURIComponent(value));
      },
      routeTemplateFinal
    );
  }

  let queryString = "";
  if (!isEmpty(queryParamsFinal)) {
    queryString = reduce(
      queryParamsFinal,
      (arr: string[], value, key) => {
        key && arr.push(`${key}=${encodeURIComponent(value)}`);
        return arr;
      },
      []
    ).join("&");
  }

  const nextLocation = union(
    trim(pathPrefixFinal, "/").split("/"),
    routeString.split("/")
  ).join("/");

  return queryString ? `/${nextLocation}?${queryString}` : `/${nextLocation}`;
};

export default { getNextLocationPath };

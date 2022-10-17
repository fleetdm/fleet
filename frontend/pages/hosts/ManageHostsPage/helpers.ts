import { isEmpty, reduce, trim, union } from "lodash";
import { buildQueryStringFromParams } from "utilities/url";

interface ILocationParams {
  pathPrefix?: string;
  routeTemplate?: string;
  routeParams?: { [key: string]: string };
  queryParams?: { [key: string]: string | number };
}

type RouteParams = Record<string, string>;

export const isAcceptableStatus = (filter: string): boolean => {
  return (
    filter === "new" ||
    filter === "online" ||
    filter === "offline" ||
    filter === "missing"
  );
};

export const isValidPolicyResponse = (filter: string): boolean => {
  return filter === "pass" || filter === "fail";
};

// Performs a grossly oversimplied validation that subject string includes substrings
// that would be expected in a textual encoding of a certificate chain per the PEM spec
// (see https://datatracker.ietf.org/doc/html/rfc7468#section-2)
// Consider using a third-party library if more robust validation is desired
export const isValidPemCertificate = (cert: string): boolean => {
  const regexPemHeader = /-----BEGIN/;
  const regexPemFooter = /-----END/;

  return regexPemHeader.test(cert) && regexPemFooter.test(cert);
};

const createRouteString = (routeTemplate: string, routeParams: RouteParams) => {
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
  return routeString;
};

export const getNextLocationPath = ({
  pathPrefix = "",
  routeTemplate = "",
  routeParams = {},
  queryParams = {},
}: ILocationParams): string => {
  const routeString = createRouteString(routeTemplate, routeParams);
  const queryString = buildQueryStringFromParams(queryParams);

  const nextLocation = union(
    trim(pathPrefix, "/").split("/"),
    routeString.split("/")
  ).join("/");

  return queryString ? `/${nextLocation}?${queryString}` : `/${nextLocation}`;
};

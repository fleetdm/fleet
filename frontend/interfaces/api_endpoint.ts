export interface IApiEndpointRef {
  method: string;
  path: string;
}

export interface IApiEndpoint extends IApiEndpointRef {
  display_name: string;
  deprecated: boolean;
}

/** Unique key for an endpoint since there's no `id` field */
export const endpointKey = (ep: IApiEndpointRef) => `${ep.method} ${ep.path}`;

/**
 * Mirrors the server's endpoint fingerprint, which grants are matched by: the
 * method's case, the path's case, and the names of path parameters don't make
 * two endpoints different.
 */
export const endpointFingerprint = (ep: IApiEndpointRef) => {
  let placeholders = 0;
  const path = ep.path
    .replace("/_version_/", "/v1/")
    .split("/")
    .map((segment) => {
      if (
        segment.startsWith(":") ||
        (segment.startsWith("{") && segment.endsWith("}"))
      ) {
        placeholders += 1;
        return `:placeholder_${placeholders}`;
      }
      return segment;
    })
    .join("/")
    .toLowerCase();
  return `${ep.method.toUpperCase()} ${path}`;
};

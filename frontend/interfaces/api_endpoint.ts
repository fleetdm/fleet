export interface IApiEndpoint {
  method: string;
  path: string;
  display_name: string;
  deprecated: boolean;
}

/** Unique key for an endpoint since there's no `id` field */
export const endpointKey = (ep: { method: string; path: string }) =>
  `${ep.method} ${ep.path}`;

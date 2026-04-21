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

import { QueryClient } from "react-query";

import { IConfig } from "interfaces/config";
import { IGlobalGoogleWorkspaceIntegration } from "interfaces/integration";

/** Builds the google_workspace PATCH payload for a new list of additional
 * domains. api_key_json is omitted so the server keeps the saved key. */
export const buildAdditionalDomainsUpdate = (
  integration: IGlobalGoogleWorkspaceIntegration,
  additionalDomains: string[]
) => ({
  integrations: {
    google_workspace: [
      {
        domain: integration.domain,
        additional_domains: additionalDomains,
        impersonated_user_email: integration.impersonated_user_email,
      },
    ],
  },
});

/** Applies a config PATCH response to the cached config. Only `integrations`
 * is taken from the response: the PATCH response leaves out fields the GET
 * returns (e.g. update_interval, vulnerabilities), and these saves only
 * change integrations. */
export const updateCachedIntegrations = (
  queryClient: QueryClient,
  updatedConfig: IConfig
) => {
  queryClient.setQueryData<IConfig | undefined>(["config"], (prev) =>
    prev ? { ...prev, integrations: updatedConfig.integrations } : prev
  );
};

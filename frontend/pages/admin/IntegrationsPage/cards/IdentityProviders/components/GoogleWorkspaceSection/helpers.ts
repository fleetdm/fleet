import { IGlobalGoogleWorkspaceIntegration } from "interfaces/integration";

/** Builds the google_workspace PATCH payload for a new list of additional
 * domains. api_key_json is omitted so the server keeps the saved key. */
const buildAdditionalDomainsUpdate = (
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

export default buildAdditionalDomainsUpdate;

import { QueryClient } from "react-query";

import { createMockConfig } from "__mocks__/configMock";
import { IConfig } from "interfaces/config";

import { updateCachedIntegrations } from "./helpers";

describe("updateCachedIntegrations", () => {
  it("takes integrations from the response and keeps fields the PATCH response omits", () => {
    const queryClient = new QueryClient();
    const cached = createMockConfig();
    queryClient.setQueryData(["config"], cached);

    const integrations = {
      ...cached.integrations,
      google_workspace: [
        {
          domain: "example.com",
          additional_domains: ["example.org"],
          impersonated_user_email: "admin@example.com",
          api_key_json: { client_email: "********" },
        },
      ],
    };
    // The PATCH response serializes these as null.
    const patchResponse = ({
      ...cached,
      integrations,
      update_interval: null,
      vulnerabilities: null,
    } as unknown) as IConfig;

    updateCachedIntegrations(queryClient, patchResponse);

    const result = queryClient.getQueryData<IConfig>(["config"]);
    expect(result?.integrations).toEqual(integrations);
    expect(result?.update_interval).toEqual(cached.update_interval);
    expect(result?.vulnerabilities).toEqual(cached.vulnerabilities);
  });

  it("leaves an empty cache empty", () => {
    const queryClient = new QueryClient();
    updateCachedIntegrations(queryClient, createMockConfig());
    expect(queryClient.getQueryData(["config"])).toBeUndefined();
  });
});

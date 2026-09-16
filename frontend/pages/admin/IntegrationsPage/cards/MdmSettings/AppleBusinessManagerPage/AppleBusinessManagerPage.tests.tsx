import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { IMdmAbToken } from "interfaces/mdm";
import mdmAbmAPI from "services/entities/mdm_apple_bm";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import AppleBusinessManagerPage from "./AppleBusinessManagerPage";

const createTestToken = (overrides: Partial<IMdmAbToken>): IMdmAbToken => ({
  id: 1,
  apple_id: "apple@example.com",
  org_name: "Acme Inc.",
  mdm_server_url: "https://example.com/mdm/apple/mdm",
  renew_date: "2027-05-29T00:00:00Z",
  terms_expired: false,
  token_invalid: false,
  default: false,
  macos_fleet: { name: "No team", fleet_id: 0 },
  ios_fleet: { name: "No team", fleet_id: 0 },
  ipados_fleet: { name: "No team", fleet_id: 0 },
  byod_fleet: { name: "No team", fleet_id: 0 },
  ...overrides,
});

describe("AppleBusinessManagerPage", () => {
  const mockConfig = createMockConfig();
  const render = createCustomRenderer({
    // sets up the react-query provider; the API layer itself is stubbed with
    // jest spies below rather than MSW handlers
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        setABMExpiry: noop,
        config: {
          ...mockConfig,
          mdm: { ...mockConfig.mdm, enabled_and_configured: true },
        },
      },
    },
  });

  it("sets a non-default token as the default and refetches the tokens", async () => {
    const getTokensSpy = jest.spyOn(mdmAbmAPI, "getTokens").mockResolvedValue({
      ab_tokens: [
        createTestToken({ id: 1, org_name: "Acme Inc.", default: true }),
        createTestToken({ id: 2, org_name: "Beta LLC" }),
      ],
    });
    const updateDefaultSpy = jest
      .spyOn(mdmAbmAPI, "updateTokenDefault")
      .mockResolvedValue({
        ab_token: createTestToken({ id: 2, org_name: "Beta LLC" }),
      });

    const { user } = render(
      <AppleBusinessManagerPage router={createMockRouter()} />
    );

    await screen.findByText("Beta LLC");
    expect(screen.getByText("Default token")).toBeInTheDocument();

    // Beta LLC sorts after Acme Inc., so its row has the second dropdown.
    await user.click(screen.getAllByText("Actions")[1]);
    await user.click(screen.getByText("Set as default token"));

    await waitFor(() => {
      expect(updateDefaultSpy).toHaveBeenCalledWith(2, true);
    });
    // initial load + refetch after the update
    await waitFor(() => {
      expect(getTokensSpy).toHaveBeenCalledTimes(2);
    });
  });

  it("unsets the default token", async () => {
    const getTokensSpy = jest.spyOn(mdmAbmAPI, "getTokens").mockResolvedValue({
      ab_tokens: [
        createTestToken({ id: 1, org_name: "Acme Inc.", default: true }),
        createTestToken({ id: 2, org_name: "Beta LLC" }),
      ],
    });
    const updateDefaultSpy = jest
      .spyOn(mdmAbmAPI, "updateTokenDefault")
      .mockResolvedValue({
        ab_token: createTestToken({ id: 1, org_name: "Acme Inc." }),
      });

    const { user } = render(
      <AppleBusinessManagerPage router={createMockRouter()} />
    );

    await screen.findByText("Acme Inc.");
    await user.click(screen.getAllByText("Actions")[0]);
    await user.click(screen.getByText("Unset default token"));

    await waitFor(() => {
      expect(updateDefaultSpy).toHaveBeenCalledWith(1, false);
    });
    expect(getTokensSpy).toHaveBeenCalled();
  });
});

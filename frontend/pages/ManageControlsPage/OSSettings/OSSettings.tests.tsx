import React from "react";
import { waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import mdmAPI from "services/entities/mdm";
import mockServer from "test/mock-server";
import {
  baseUrl,
  createCustomRenderer,
  createMockRouter,
} from "test/test-utils";

import OSSettings from "./OSSettings";

const baseProps = {
  router: createMockRouter(),
  currentPage: 0,
  params: { section: "disk-encryption" },
  location: { search: "?fleet_id=5" },
};

// Verifies the gate that keeps OSSettings from firing team-scoped queries
// against the wrong fleet on refresh, before useTeamIdParam in the parent
// has resolved the URL to an available fleet.
describe("OSSettings", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isPremiumTier: true,
        isGlobalAdmin: true,
        config: { mdm: { enabled_and_configured: true } },
      },
    },
  });

  // The DiskEncryption card (default section) hits /fleets/:id — return an
  // empty team config so MSW doesn't log unhandled-request warnings.
  beforeEach(() => {
    mockServer.use(
      http.get(baseUrl("/fleets/:id"), () =>
        HttpResponse.json({ team: { mdm: {} } })
      )
    );
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("does not fetch the profile status summary while teamIdForApi is undefined", async () => {
    const summarySpy = jest
      .spyOn(mdmAPI, "getProfilesStatusSummary")
      .mockResolvedValue({ verified: 0, verifying: 0, pending: 0, failed: 0 });

    render(<OSSettings {...baseProps} teamIdForApi={undefined} />);

    // Give react-query a tick — a premature fetch would kick off here.
    await new Promise((r) => setTimeout(r, 50));

    expect(summarySpy).not.toHaveBeenCalled();
  });

  it("fetches the profile status summary for the passed teamIdForApi", async () => {
    const summarySpy = jest
      .spyOn(mdmAPI, "getProfilesStatusSummary")
      .mockResolvedValue({ verified: 0, verifying: 0, pending: 0, failed: 0 });

    render(<OSSettings {...baseProps} teamIdForApi={5} />);

    await waitFor(() => {
      expect(summarySpy).toHaveBeenCalledWith(5);
    });
    expect(summarySpy).toHaveBeenCalledTimes(1);
  });
});

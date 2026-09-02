import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { createCustomRenderer, createMockRouter } from "test/test-utils";
import mockServer from "test/mock-server";
import { customDeviceSoftwareHandler } from "test/handlers/device-handler";
import { createMockDeviceSoftware } from "__mocks__/deviceUserMock";
import { noop } from "lodash";

import { HostPlatform } from "interfaces/platform";

import HostSoftware, { parseHostSoftwareQueryParams } from "./HostSoftware";

const mockRouter = createMockRouter();

// The My device page is token-authenticated and has no app session, so the
// premium tier is passed in as a prop rather than read from the app context.
// These tests verify that the CVSS severity filter dialogue is shown/hidden on
// the My device page based on that prop.
describe("HostSoftware - My device page filters", () => {
  const baseProps = {
    id: "device-token",
    platform: "windows" as HostPlatform,
    router: mockRouter,
    queryParams: parseHostSoftwareQueryParams({}),
    pathname: "/device/device-token/software",
    hostTeamId: 0,
    onShowInventoryVersions: noop,
    isSoftwareEnabled: true,
    isMyDevicePage: true,
  };

  const renderDeviceSoftware = (props = {}) =>
    createCustomRenderer({ withBackendMock: true })(
      <HostSoftware {...baseProps} {...props} />
    );

  beforeEach(() => {
    // Return at least one software item so the filter button is enabled.
    mockServer.use(
      customDeviceSoftwareHandler({
        count: 1,
        software: [createMockDeviceSoftware()],
      })
    );
  });

  it("shows the CVSS severity filter fields for premium My device users", async () => {
    const { user } = renderDeviceSoftware({ isPremiumTier: true });

    const filterButton = await screen.findByRole("button", {
      name: /filter/i,
    });
    await user.click(filterButton);

    // Premium fields are present in the filters modal.
    expect(screen.getByText(/Vulnerable software/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Min score/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Max score/i)).toBeInTheDocument();
    expect(screen.getByText(/Has known exploit/i)).toBeInTheDocument();
  });

  it("hides the CVSS severity filter fields for free My device users", async () => {
    const { user } = renderDeviceSoftware({ isPremiumTier: false });

    const filterButton = await screen.findByRole("button", {
      name: /filter/i,
    });
    await user.click(filterButton);

    // Only the vulnerable software toggle is shown; premium fields are absent.
    await waitFor(() => {
      expect(screen.getByText(/Vulnerable software/i)).toBeInTheDocument();
    });
    expect(screen.queryByLabelText(/Min score/i)).not.toBeInTheDocument();
    expect(screen.queryByLabelText(/Max score/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Has known exploit/i)).not.toBeInTheDocument();
  });
});

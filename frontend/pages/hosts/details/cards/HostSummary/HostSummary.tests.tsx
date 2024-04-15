import React from "react";
import { noop } from "lodash";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockHostSummary } from "__mocks__/hostMock";

import HostSummary from "./HostSummary";

describe("Host Actions Dropdown", () => {
  describe("Agent data", () => {
    it("with all info present, render Agent header with orbit_version and tooltip with all 3 data points", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary();
      const orbitVersion = summaryData.orbit_version as string;
      const osqueryVersion = summaryData.osquery_version as string;
      const fleetdVersion = summaryData.fleet_desktop_version as string;

      const { user } = render(
        <HostSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchHost={noop}
          renderActionButtons={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();
      await user.hover(screen.getByText(new RegExp(orbitVersion, "i")));

      expect(
        screen.getByText(new RegExp(osqueryVersion, "i"))
      ).toBeInTheDocument();
      expect(
        screen.getByText(new RegExp(fleetdVersion, "i"))
      ).toBeInTheDocument();
    });

    it("omit fleet desktop from tooltip if no fleet desktop version", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({
        fleet_desktop_version: null,
      });
      const orbitVersion = summaryData.orbit_version as string;
      const osqueryVersion = summaryData.osquery_version as string;

      const { user } = render(
        <HostSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchHost={noop}
          renderActionButtons={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();
      await user.hover(screen.getByText(new RegExp(orbitVersion, "i")));

      expect(
        screen.getByText(new RegExp(osqueryVersion, "i"))
      ).toBeInTheDocument();
      expect(screen.queryByText(/Fleet desktop:/i)).not.toBeInTheDocument();
    });

    it("for Chromebooks, render Agent header with osquery_version that is the fleetd chrome version and no tooltip", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({
        platform: "chrome",
        osquery_version: "fleetd-chrome 1.2.0",
      });

      const fleetdChromeVersion = summaryData.osquery_version as string;

      const { user } = render(
        <HostSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchHost={noop}
          renderActionButtons={() => null}
        />
      );

      expect(screen.getByText("Agent")).toBeInTheDocument();
      await user.hover(screen.getByText(new RegExp(fleetdChromeVersion, "i")));
      expect(screen.queryByText("Osquery")).not.toBeInTheDocument();
    });
    it("for non-Chromebooks with no orbit_version, render Osquery header with osquery_version and no tooltip", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({
        orbit_version: null,
      });

      const osqueryVersion = summaryData.osquery_version as string;

      const { user } = render(
        <HostSummary
          summaryData={summaryData}
          showRefetchSpinner={false}
          onRefetchHost={noop}
          renderActionButtons={() => null}
        />
      );

      expect(screen.getByText("Osquery")).toBeInTheDocument();
      await user.hover(screen.getByText(new RegExp(osqueryVersion, "i")));
      expect(screen.queryByText(/Orbit/i)).not.toBeInTheDocument();
      expect(screen.queryByText(/Fleet desktop/i)).not.toBeInTheDocument();
    });
  });
});

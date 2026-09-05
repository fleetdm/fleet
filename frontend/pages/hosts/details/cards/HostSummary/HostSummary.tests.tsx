import React from "react";
import { screen, within } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockHostSummary } from "__mocks__/hostMock";

import { BootstrapPackageStatus } from "interfaces/mdm";
import { HostPlatform } from "interfaces/platform";
import HostSummary from "./HostSummary";

describe("Host Summary section", () => {
  describe("Issues data", () => {
    it("omit issues header if no issues", async () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({});

      render(<HostSummary summaryData={summaryData} />);

      expect(screen.queryByText("Issues")).not.toBeInTheDocument();
    });
  });

  describe("Fleet data", () => {
    it("renders the team name when present", () => {
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({ team_name: "Engineering" });
      render(<HostSummary summaryData={summaryData} isPremiumTier />);
      expect(screen.getByText("Fleet").nextElementSibling).toHaveTextContent(
        "Engineering"
      );
    });

    it("renders 'No team' when team_name is '---'", () => {
      const render = createCustomRenderer({
        /* ...context... */
      });
      const summaryData = createMockHostSummary({ team_name: "---" });
      render(<HostSummary summaryData={summaryData} isPremiumTier />);
      expect(screen.getByText("Unassigned")).toBeInTheDocument();
    });
  });

  describe("iOS and iPadOS data", () => {
    it("for iOS, renders Fleet data only", async () => {
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
        team_id: 2,
        team_name: "Mobile",
        platform: "ios",
        os_version: "iOS 14.7.1",
      });

      const teamName = summaryData.team_name as string;

      render(<HostSummary summaryData={summaryData} isPremiumTier />);

      expect(screen.getByText("Fleet").nextElementSibling).toHaveTextContent(
        teamName
      );
    });
    it("for iPadOS, renders Fleet data only", async () => {
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
        team_id: 2,
        team_name: "Mobile",
        platform: "ipados",
        os_version: "iPadOS 16.7.8",
      });

      const teamName = summaryData.team_name as string;

      render(<HostSummary summaryData={summaryData} isPremiumTier />);

      expect(screen.getByText("Fleet").nextElementSibling).toHaveTextContent(
        teamName
      );
    });
  });

  describe("Maintenance window data", () => {
    it("renders maintenance window data with timezone", async () => {
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
        maintenance_window: {
          starts_at: "3025-06-24T20:48:14-03:00",
          timezone: "America/Argentina/Buenos_Aires",
        },
      });
      const prettyStartTime = /Jun 24 at 8:48 PM/;

      render(<HostSummary summaryData={summaryData} isPremiumTier />);

      expect(screen.getByText("Scheduled maintenance")).toBeInTheDocument();
      expect(screen.getByText(prettyStartTime)).toBeInTheDocument();
    });
  });

  describe("Mobile Status row", () => {
    it.each<[string, HostPlatform, string]>([
      ["Android", "android", "Android 14"],
      ["iOS", "ios", "iOS 17.4"],
      ["iPadOS", "ipados", "iPadOS 17.4"],
    ])(
      "renders the Status row for a Free-tier %s host now that the API returns real online/offline for mobile",
      (_label, platform, os_version) => {
        const render = createCustomRenderer({
          context: {
            app: {
              isPremiumTier: false,
              isGlobalAdmin: true,
              currentUser: createMockUser(),
            },
          },
        });
        const summaryData = createMockHostSummary({
          platform,
          os_version,
          status: "online",
        });

        render(<HostSummary summaryData={summaryData} isPremiumTier={false} />);

        expect(screen.getByText("Status")).toBeInTheDocument();
        expect(screen.getByText("Online")).toBeInTheDocument();
      }
    );

    it("wraps the Status label in an explainer tooltip only for iOS/iPadOS", () => {
      const renderMobile = (platform: HostPlatform) => {
        const render = createCustomRenderer({
          context: {
            app: { isPremiumTier: true, currentUser: createMockUser() },
          },
        });
        const summaryData = createMockHostSummary({
          platform,
          status: "online",
        });
        return render(<HostSummary summaryData={summaryData} isPremiumTier />);
      };

      // The Status <dt> title is what wraps in TooltipWrapper for iOS/iPadOS.
      // Scope the query to the title element (scoped to `container` since two
      // renders happen in this test) to avoid matching the pill's own
      // TooltipWrapper on the <dd> value side or the other render's DOM.
      const titleHasTooltip = (root: HTMLElement) => {
        const statusTitle = within(root).getByText("Status").closest("dt");
        return !!statusTitle?.querySelector(".component__tooltip-wrapper");
      };

      const { container: iosContainer } = renderMobile("ios");
      expect(titleHasTooltip(iosContainer)).toBe(true);

      const { container: androidContainer } = renderMobile("android");
      expect(titleHasTooltip(androidContainer)).toBe(false);
    });
  });

  describe("Bootstrap package data", () => {
    it("renders Bootstrap package indicator when status is present", () => {
      const toggleBootstrapPackageModal = jest.fn();
      const render = createCustomRenderer({
        context: {
          app: {
            isPremiumTier: true,
            isGlobalAdmin: true,
            currentUser: createMockUser(),
          },
        },
      });
      const summaryData = createMockHostSummary({ platform: "darwin" });
      const bootstrapPackageData = {
        status: "installed" as BootstrapPackageStatus,
      };
      render(
        <HostSummary
          summaryData={summaryData}
          bootstrapPackageData={bootstrapPackageData}
          toggleBootstrapPackageModal={toggleBootstrapPackageModal}
        />
      );
      expect(screen.getByText("Bootstrap package")).toBeInTheDocument();
    });
  });
});

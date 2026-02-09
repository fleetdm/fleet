import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer } from "test/test-utils";

import createMockUser from "__mocks__/userMock";
import { createMockHostSummary } from "__mocks__/hostMock";

import { BootstrapPackageStatus } from "interfaces/mdm";
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

  describe("Team data", () => {
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
      expect(screen.getByText("Team").nextElementSibling).toHaveTextContent(
        "Engineering"
      );
    });

    it("renders 'No team' when team_name is '---'", () => {
      const render = createCustomRenderer({
        /* ...context... */
      });
      const summaryData = createMockHostSummary({ team_name: "---" });
      render(<HostSummary summaryData={summaryData} isPremiumTier />);
      expect(screen.getByText("No team")).toBeInTheDocument();
    });
  });

  describe("iOS and iPadOS data", () => {
    it("for iOS, renders Team data only", async () => {
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

      expect(screen.getByText("Team").nextElementSibling).toHaveTextContent(
        teamName
      );
    });
    it("for iPadOS, renders Team data only", async () => {
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

      expect(screen.getByText("Team").nextElementSibling).toHaveTextContent(
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

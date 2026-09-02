import React from "react";
import { render, screen } from "@testing-library/react";

import { PlatformValueOptions } from "utilities/constants";

import MetricsHostCounts from "./MetricsHostCounts";

// Render react-router's <Link> as a plain anchor so the summary cards
// (which wrap themselves in a linkable <Card>) can render without a
// surrounding <Router>.
jest.mock("react-router", () => ({
  Link: ({ to, children }: { to: string; children: React.ReactNode }) => (
    <a href={to}>{children}</a>
  ),
}));

const TOTAL_HOSTS_TITLE = "Total hosts";
const MISSING_HOSTS_TITLE = "Missing hosts";
const LOW_DISK_SPACE_TITLE = "Low disk space hosts";
const ABM_ISSUE_TITLE = "AB issue";

interface IRenderCase {
  platform: PlatformValueOptions;
  totalHosts: boolean;
  missingHosts: boolean;
  lowDiskSpaceHosts: boolean;
}

const renderMetrics = ({
  platform,
  isPremiumTier,
  abmIssueCount = 0,
}: {
  platform: PlatformValueOptions;
  isPremiumTier: boolean;
  abmIssueCount?: number;
}) =>
  render(
    <MetricsHostCounts
      currentTeamId={undefined}
      selectedPlatform={platform}
      totalHostCount={42}
      isPremiumTier={isPremiumTier}
      missingCount={3}
      lowDiskSpaceCount={5}
      abmIssueCount={abmIssueCount}
      selectedPlatformLabelId={undefined}
    />
  );

const expectCards = ({
  totalHosts,
  missingHosts,
  lowDiskSpaceHosts,
}: Omit<IRenderCase, "platform">) => {
  expect(!!screen.queryByText(TOTAL_HOSTS_TITLE)).toBe(totalHosts);
  expect(!!screen.queryByText(MISSING_HOSTS_TITLE)).toBe(missingHosts);
  expect(!!screen.queryByText(LOW_DISK_SPACE_TITLE)).toBe(lowDiskSpaceHosts);
};

// Missing hosts renders on both tiers; Low disk space is Premium-only.
// Both are hidden on iOS, iPadOS, and Android (no missing/low-disk-space
// data model for those platforms). Total hosts renders only on "all".
describe("MetricsHostCounts", () => {
  describe("Premium tier", () => {
    const premiumCases: IRenderCase[] = [
      {
        platform: "all",
        totalHosts: true,
        missingHosts: true,
        lowDiskSpaceHosts: true,
      },
      {
        platform: "darwin",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: true,
      },
      {
        platform: "windows",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: true,
      },
      {
        platform: "linux",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: true,
      },
      {
        platform: "chrome",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: true,
      },
      {
        platform: "ios",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "ipados",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "android",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
    ];

    it.each(premiumCases)(
      "$platform: shows Total=$totalHosts, Missing=$missingHosts, LowDisk=$lowDiskSpaceHosts",
      ({ platform, ...expected }) => {
        renderMetrics({ platform, isPremiumTier: true });
        expectCards(expected);
      }
    );
  });

  describe("Free tier", () => {
    // Free never shows Low disk space (Premium-only). Missing hosts renders
    // on the same platforms as Premium.
    const freeCases: IRenderCase[] = [
      {
        platform: "all",
        totalHosts: true,
        missingHosts: true,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "darwin",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "windows",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "linux",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "chrome",
        totalHosts: false,
        missingHosts: true,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "ios",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "ipados",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
      {
        platform: "android",
        totalHosts: false,
        missingHosts: false,
        lowDiskSpaceHosts: false,
      },
    ];

    it.each(freeCases)(
      "$platform: shows Total=$totalHosts, Missing=$missingHosts, LowDisk=$lowDiskSpaceHosts",
      ({ platform, ...expected }) => {
        renderMetrics({ platform, isPremiumTier: false });
        expectCards(expected);
      }
    );
  });

  describe("ABM issue card", () => {
    it("renders on Premium when abmIssueCount > 0", () => {
      renderMetrics({
        platform: "darwin",
        isPremiumTier: true,
        abmIssueCount: 2,
      });
      expect(screen.getByText(ABM_ISSUE_TITLE)).toBeInTheDocument();
    });

    it("does not render when abmIssueCount is 0", () => {
      renderMetrics({
        platform: "darwin",
        isPremiumTier: true,
        abmIssueCount: 0,
      });
      expect(screen.queryByText(ABM_ISSUE_TITLE)).not.toBeInTheDocument();
    });

    // DashboardPage never sets abmIssueCount on Free, but the component
    // itself doesn't tier-gate this card — protecting that invariant here.
    it("guards purely on count, not tier (Free with count > 0 would render, but the parent never populates it)", () => {
      renderMetrics({
        platform: "darwin",
        isPremiumTier: false,
        abmIssueCount: 2,
      });
      expect(screen.getByText(ABM_ISSUE_TITLE)).toBeInTheDocument();
    });
  });
});

import { screen } from "@testing-library/react";
import React from "react";

import {
  createMockConfig,
  createMockMdmConfig,
  DEFAULT_LICENSE_MOCK,
} from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import MainContent from "./MainContent";

const GRAPH_BANNER_TEXT =
  "Your Microsoft Graph client secret is expired, deleted, or has missing " +
  "permissions. Windows Autopilot devices won't sync to Fleet as pending " +
  "hosts. Users with the admin role in Fleet can update the credential.";

const renderMainContent = ({
  credentialInvalid = false,
  isVppExpired = false,
} = {}) => {
  const render = createCustomRenderer({
    context: {
      // Every banner but the APNs one sits inside MainContent's premium gate; these all assume premium.
      app: {
        isPremiumTier: true,
        isVppExpired,
        config: createMockConfig({
          mdm: createMockMdmConfig({
            microsoft_graph_credential_invalid: credentialInvalid,
          }),
        }),
      },
    },
  });

  return render(
    <MainContent>
      <div>content</div>
    </MainContent>
  );
};

describe("MainContent app-wide banners", () => {
  it("shows the Microsoft Graph banner when the credential is invalid", () => {
    renderMainContent({ credentialInvalid: true });
    expect(screen.getByText(GRAPH_BANNER_TEXT)).toBeInTheDocument();
  });

  it("does not show the banner when the credential is healthy", () => {
    renderMainContent({ credentialInvalid: false });
    expect(screen.queryByText(GRAPH_BANNER_TEXT)).not.toBeInTheDocument();
  });

  it("yields to the VPP banner, which sits above it in the priority order", () => {
    // Only one banner renders at a time.
    renderMainContent({ credentialInvalid: true, isVppExpired: true });
    expect(screen.queryByText(GRAPH_BANNER_TEXT)).not.toBeInTheDocument();
    expect(
      screen.getByText(/Volume Purchasing Program \(VPP\) content token/i)
    ).toBeInTheDocument();
  });
});

describe("MainContent license host limit banner", () => {
  const renderWithHostCount = ({
    hostCount,
    deviceCount = 50,
    isPremiumTier = true,
  }: {
    hostCount?: number;
    deviceCount?: number;
    isPremiumTier?: boolean;
  }) => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier,
          hostCount,
          config: createMockConfig({
            license: {
              ...DEFAULT_LICENSE_MOCK,
              // The default mock license is expired, which would show the expiration banner instead.
              expiration: "2099-01-01T00:00:00Z",
              device_count: deviceCount,
            },
          }),
        },
      },
    });
    return render(
      <MainContent>
        <div>content</div>
      </MainContent>
    );
  };

  it("shows the banner when hosts exceed the licensed count", () => {
    renderWithHostCount({ hostCount: 61 });
    expect(
      screen.getByText(/Your license covers 50 hosts, but you have 61 enrolled/)
    ).toBeInTheDocument();
  });

  it("does not show the banner until hosts are more than 20% over the licensed count", () => {
    renderWithHostCount({ hostCount: 60 });
    expect(screen.queryByText(/Your license covers/)).not.toBeInTheDocument();
  });

  it("does not show the banner when the host count isn't loaded", () => {
    renderWithHostCount({ hostCount: undefined });
    expect(screen.queryByText(/Your license covers/)).not.toBeInTheDocument();
  });

  it("does not show the banner when the license has no host limit", () => {
    renderWithHostCount({ hostCount: 61, deviceCount: 0 });
    expect(screen.queryByText(/Your license covers/)).not.toBeInTheDocument();
  });

  it("does not show the banner on Fleet Free", () => {
    renderWithHostCount({ hostCount: 61, isPremiumTier: false });
    expect(screen.queryByText(/Your license covers/)).not.toBeInTheDocument();
  });
});

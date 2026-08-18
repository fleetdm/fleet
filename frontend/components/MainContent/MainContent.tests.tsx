import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";

import MainContent from "./MainContent";

const GRAPH_BANNER_TEXT = /Microsoft Graph client secret is invalid/i;

const renderMainContent = ({
  credentialInvalid = false,
  isPremiumTier = true,
  isVppExpired = false,
} = {}) => {
  const render = createCustomRenderer({
    context: {
      app: {
        isPremiumTier,
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

  it("does not show the banner on Fleet Free", () => {
    // Every banner but the APNs one is premium-only.
    renderMainContent({ credentialInvalid: true, isPremiumTier: false });

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

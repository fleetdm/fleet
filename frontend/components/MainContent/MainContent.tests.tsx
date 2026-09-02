import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";

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

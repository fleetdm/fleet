import React from "react";
import { screen } from "@testing-library/react";

import { createMockConfig } from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";

import IdentityProviders from "./IdentityProviders";

const renderWith = () =>
  createCustomRenderer({
    withBackendMock: true,
    context: { app: { config: createMockConfig() } },
  });

describe("IdentityProviders", () => {
  it("gates the whole card behind premium for non-premium tiers", () => {
    const render = renderWith();
    render(
      <IdentityProviders appConfig={createMockConfig()} isPremiumTier={false} />
    );

    expect(
      screen.getByText("This feature is included in Fleet Premium.")
    ).toBeInTheDocument();
    // The section title stays above the premium message (matching other sections).
    expect(screen.getByText("Identity provider (IdP)")).toBeInTheDocument();
    // The Google Workspace section does not render when not premium.
    expect(screen.queryByText("Google Workspace")).not.toBeInTheDocument();
  });
});

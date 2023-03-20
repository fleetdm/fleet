import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import IntegrationsPage from "./IntegrationsPage";

describe("Integrations Page", () => {
  it("renders the MDM section in the side nav if MDM feature is enabled", () => {
    const render = createCustomRenderer({
      withBackendMock: true,
      context: {
        app: { isMdmFeatureFlagEnabled: true },
      },
    });

    render(<IntegrationsPage params={{ section: "mdm" }} />);

    expect(
      screen.getByText("Mobile device management (MDM)")
    ).toBeInTheDocument();
  });
});

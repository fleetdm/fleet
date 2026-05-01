import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const router = createMockRouter();

describe("SoftwareAppStoreAndroid", () => {
  it("shows enable Android MDM message when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText("To add Android apps, first enable Android MDM.")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Enable Android MDM" })
    ).toBeInTheDocument();
  });

  it("shows the Android form when Android MDM is enabled", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isAndroidMdmEnabledAndConfigured: true,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(
      screen.queryByText("Android MDM isn't enabled")
    ).not.toBeInTheDocument();
  });
});

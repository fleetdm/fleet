import React from "react";
import { screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import SoftwareAppStoreAndroid from "./SoftwareAppStoreAndroid";

const router = createMockRouter();

describe("SoftwareAppStoreAndroid", () => {
  it("shows enable button for admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
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

  it("shows ask your admin copy for non-admins when Android MDM is not configured", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: false,
          isAnyTeamAdmin: false,
          isAndroidMdmEnabledAndConfigured: false,
        },
      },
    });

    render(<SoftwareAppStoreAndroid currentTeamId={1} router={router} />);

    expect(screen.getByText("Android MDM isn't enabled")).toBeInTheDocument();
    expect(
      screen.getByText(
        "To add Android apps, ask your admin to enable Android MDM."
      )
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Enable Android MDM" })
    ).not.toBeInTheDocument();
  });

  it("shows the Android form when Android MDM is enabled", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          isGlobalAdmin: true,
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

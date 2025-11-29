import React from "react";

import { screen } from "@testing-library/react";

import { createMockRouter, createCustomRenderer } from "test/test-utils";
import { createMockConfig, createMockMdmConfig } from "__mocks__/configMock";
import WindowsMdmPage from "./WindowsMdmPage";

describe("WindowsMdmPage", () => {
  it("renders only the windows mdm slider and description when on free tier", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: false,
          config: createMockConfig(),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    // switch and description only shown
    expect(screen.getByRole("switch")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Turns on MDM for Windows hosts that enroll to Fleet (excluding servers)."
      )
    ).toBeInTheDocument();

    // no end user experience form
    expect(screen.queryByLabelText("Automatic")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Manual")).not.toBeInTheDocument();
  });

  it("renders the end user experience form as disabled when MDM is off", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({ windows_enabled_and_configured: false }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    expect(screen.getByLabelText("Automatic")).toBeDisabled();
    expect(screen.getByLabelText("Manual")).toBeDisabled();
  });

  it("renders the automatically migrate checkbox if automatic mdm enrollment is selected", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          isPremiumTier: true,
          config: createMockConfig({
            mdm: createMockMdmConfig({
              enable_turn_on_windows_mdm_manually: false,
              windows_enabled_and_configured: true,
            }),
          }),
        },
      },
    });

    render(<WindowsMdmPage router={createMockRouter()} />);

    // automatic is selected and the checkbox is visible
    expect(screen.getByLabelText("Automatic")).toBeChecked();
    expect(screen.getByRole("checkbox")).toBeVisible();
  });
});

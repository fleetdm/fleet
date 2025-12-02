import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import PATHS from "router/paths";

import GenericMsgWithNavButton from "./GenericMsgWithNavButton";

describe("GenericMsgWithNavButton", () => {
  it("renders with passed in header and info", () => {
    render(
      <GenericMsgWithNavButton
        header="Manage your hosts"
        info="MDM must be turned on to change settings on your hosts."
        path={PATHS.ADMIN_INTEGRATIONS_MDM}
        buttonText="Turn on"
        router={createMockRouter()}
      />
    );

    expect(screen.getByText("Manage your hosts")).toBeInTheDocument();
    expect(
      screen.getByText(
        "MDM must be turned on to change settings on your hosts."
      )
    ).toBeInTheDocument();
  });

  it('renders "Turn on" button for global admin pushes to /settings/integrration/mdm when "Turn on" button is clicked', () => {
    const customRender = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
        },
      },
    });

    customRender(
      <GenericMsgWithNavButton
        header="test"
        info="test"
        buttonText="Turn on"
        path={PATHS.ADMIN_INTEGRATIONS_MDM}
        router={createMockRouter()}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Turn on" }));

    expect(createMockRouter().push).toHaveBeenCalledWith(
      PATHS.ADMIN_INTEGRATIONS_MDM
    );
  });

  it("does not render the button for non-global admin", () => {
    const customRender = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: false,
        },
      },
    });

    customRender(
      <GenericMsgWithNavButton
        header="test"
        info="test"
        path={"test"}
        buttonText="Turn on"
        router={createMockRouter()}
      />
    );

    expect(screen.queryByText("Turn on")).not.toBeInTheDocument();
  });
});

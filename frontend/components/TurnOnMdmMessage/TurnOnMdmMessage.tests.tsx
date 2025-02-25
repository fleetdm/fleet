import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import TurnOnMdmMessage from "./TurnOnMdmMessage";

describe("TurnOnMdmMessage", () => {
  it("renders with default header and info", () => {
    render(<TurnOnMdmMessage router={createMockRouter()} />);

    expect(screen.getByText("Manage your hosts")).toBeInTheDocument();
    expect(
      screen.getByText(
        "MDM must be turned on to change settings on your hosts."
      )
    ).toBeInTheDocument();
  });

  it("renders with custom header and info", () => {
    render(
      <TurnOnMdmMessage
        router={createMockRouter()}
        header="Custom header"
        info="Custom info"
      />
    );

    expect(screen.getByText("Custom header")).toBeInTheDocument();
    expect(screen.getByText("Custom info")).toBeInTheDocument();
  });

  it('renders "Turn on" button for global admin pushes to /settings/integrration/mdm when "Turn on" button is clicked', () => {
    const customRender = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: true,
        },
      },
    });

    customRender(<TurnOnMdmMessage router={createMockRouter()} />);

    fireEvent.click(screen.getByText("Turn on"));
    expect(createMockRouter().push).toHaveBeenCalledWith(
      "/settings/integrations/mdm"
    );
  });

  it('does not render "Turn on" button for non-global admin', () => {
    const customRender = createCustomRenderer({
      context: {
        app: {
          isGlobalAdmin: false,
        },
      },
    });

    customRender(<TurnOnMdmMessage router={createMockRouter()} />);

    expect(screen.queryByText("Turn on")).not.toBeInTheDocument();
  });
});

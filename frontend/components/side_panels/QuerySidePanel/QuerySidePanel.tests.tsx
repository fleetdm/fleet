import React from "react";
import { noop } from "lodash";
import { render, screen, fireEvent } from "@testing-library/react";

import createMockOsqueryTable from "__mocks__/osqueryTableMock";
import QuerySidePanel from "./QuerySidePanel";

describe("QuerySidePanel - component", () => {
  it("renders the query side panel with the correct table selected", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const tableDropdownText = screen.getByDisplayValue(/users/i);
    expect(tableDropdownText).toBeInTheDocument();
  });

  it("renders platform compatibility", () => {
    const { container } = render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const platformList = container.getElementsByClassName("platform-list-item");
    const platformCompatibility = screen.getByTestId("compatibility");

    expect(platformList.length).toBe(4);
    expect(platformCompatibility).toHaveTextContent(/macos/i);
    expect(platformCompatibility).toHaveTextContent(/windows/i);
    expect(platformCompatibility).toHaveTextContent(/linux/i);
    expect(platformCompatibility).toHaveTextContent(/chromeos/i);
  });

  it("renders the correct number of columns including hiding columns set to hidden", () => {
    const { container } = render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const platformList = container.getElementsByClassName("column-list-item");
    expect(platformList.length).toBe(11); // 2 columns are set to hidden
  });

  // commmenting out the below test as the new tooltip is now event-based and doesn't exist in the
  // DOM at this point. TODO(jacob) – figure out a work around

  it("renders the platform specific column tooltip", async () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    // await screen.getByText("email").onmouseenter;

    // const el = screen.getByText("email");
    // await user.hover();
    await fireEvent.mouseEnter(screen.getByText("email"));

    const tooltip = screen.getByText(/only available on chrome/i);
    expect(tooltip).toBeInTheDocument();
  });

  it("render an example", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const exampleHeader = screen.getByText(
      /List users that have interactive access via a shell that isn't false/i
    );
    const example = screen.getByText("Example");

    expect(exampleHeader).toBeInTheDocument();
    expect(example).toBeInTheDocument();
  });
  it("render notes", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable({
          notes: "This table is being used for testing.",
        })}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const notesHeader = screen.getByText(/Notes/i);
    const notesText = screen.getByText(/This table is being used for testing/i);

    expect(notesHeader).toBeInTheDocument();
    expect(notesText).toBeInTheDocument();
  });
  it("renders a link to the source", () => {
    render(
      <QuerySidePanel
        selectedOsqueryTable={createMockOsqueryTable()}
        onOsqueryTableSelect={() => noop}
        onClose={noop}
      />
    );

    const text = screen.getByText("Source");
    const icon = screen.queryByTestId("Icon");

    expect(text).toBeInTheDocument();
    expect(icon).toBeNull();
    expect(text.closest("a")).toHaveAttribute(
      "href",
      "https://www.fleetdm.com/tables/users"
    );
    expect(text.closest("a")).toHaveAttribute("target", "_blank");
  });
});

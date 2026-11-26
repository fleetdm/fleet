import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import ActivityTypeDropdown from "./ActivityTypeDropdown";

describe("ActivityTypeDropdown", () => {
  it("lists the viewed unlock PIN activity type", async () => {
    const renderer = createCustomRenderer({ withBackendMock: true });
    const { user } = renderer(
      <ActivityTypeDropdown value="" onSelect={noop} />
    );

    await user.click(screen.getByText("All types"));

    expect(screen.getByText("Viewed unlock PIN")).toBeInTheDocument();
  });

  it("focuses the search input when opened so typing filters immediately", async () => {
    const user = userEvent.setup();
    render(<ActivityTypeDropdown value="" onSelect={noop} />);

    await user.click(screen.getByText("All types"));
    const searchInput = screen.getByRole("textbox", {
      name: "Search activity types",
    });
    expect(searchInput).toHaveFocus();

    await user.keyboard("unlock PIN");
    expect(searchInput).toHaveValue("unlock PIN");
    expect(screen.getByText("Viewed unlock PIN")).toBeInTheDocument();
    expect(screen.queryByText("Wiped host")).not.toBeInTheDocument();
  });

  it("closes the menu on click outside while the search input is focused", async () => {
    const user = userEvent.setup();
    render(
      <div>
        <button type="button">Outside</button>
        <ActivityTypeDropdown value="" onSelect={noop} />
      </div>
    );

    await user.click(screen.getByText("All types"));
    const searchInput = screen.getByRole("textbox", {
      name: "Search activity types",
    });
    await user.click(searchInput);
    expect(searchInput).toHaveFocus();

    await user.click(screen.getByRole("button", { name: "Outside" }));
    expect(
      screen.queryByRole("textbox", { name: "Search activity types" })
    ).not.toBeInTheDocument();
  });

  it("selects a filtered option with the keyboard from the search input", async () => {
    const user = userEvent.setup();
    const onSelect = jest.fn();
    render(<ActivityTypeDropdown value="" onSelect={onSelect} />);

    await user.click(screen.getByText("All types"));
    await user.keyboard("viewed unlock PIN");
    await user.keyboard("{Enter}");

    expect(onSelect).toHaveBeenCalledWith("viewed_host_unlock_pin");
  });

  it("focuses the search input when reopened after selecting an option", async () => {
    const user = userEvent.setup();
    const Wrapper = () => {
      const [value, setValue] = React.useState("");
      return <ActivityTypeDropdown value={value} onSelect={setValue} />;
    };
    render(<Wrapper />);

    await user.click(screen.getByText("All types"));
    await user.click(screen.getByText("Wiped host"));
    expect(
      screen.queryByRole("textbox", { name: "Search activity types" })
    ).not.toBeInTheDocument();

    await user.click(screen.getByText("Wiped host"));
    const searchInput = screen.getByRole("textbox", {
      name: "Search activity types",
    });
    expect(searchInput).toHaveFocus();
  });

  it.each([
    ["clicking outside", "click-outside"],
    ["pressing Escape", "escape"],
    ["clicking the dropdown again", "toggle"],
  ])("clears the search text after closing by %s", async (_, closeBy) => {
    const user = userEvent.setup();
    render(
      <div>
        <button type="button">Outside</button>
        <ActivityTypeDropdown value="" onSelect={noop} />
      </div>
    );

    await user.click(screen.getByText("All types"));
    await user.keyboard("wiped");
    expect(screen.queryByText("Added App Store app")).not.toBeInTheDocument();

    if (closeBy === "click-outside") {
      await user.click(screen.getByRole("button", { name: "Outside" }));
    } else if (closeBy === "escape") {
      await user.keyboard("{Escape}");
    } else {
      await user.click(screen.getByText("All types"));
    }
    expect(
      screen.queryByRole("textbox", { name: "Search activity types" })
    ).not.toBeInTheDocument();

    await user.click(screen.getByText("All types"));
    expect(
      screen.getByRole("textbox", { name: "Search activity types" })
    ).toHaveValue("");
    expect(screen.getByText("Added App Store app")).toBeInTheDocument();
  });
});

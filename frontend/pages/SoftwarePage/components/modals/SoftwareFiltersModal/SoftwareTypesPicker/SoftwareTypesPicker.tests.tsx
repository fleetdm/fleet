import { screen, waitFor } from "@testing-library/react";
import React, { useState } from "react";

import { SOFTWARE_TYPES } from "interfaces/software";
import { renderWithSetup } from "test/test-utils";

import SoftwareTypesPicker from "./SoftwareTypesPicker";

const ControlledPicker = ({ initialKeys = [] }: { initialKeys?: string[] }) => {
  const [keys, setKeys] = useState(initialKeys);
  return (
    <SoftwareTypesPicker
      availableTypes={SOFTWARE_TYPES}
      selectedKeys={keys}
      onChange={setKeys}
    />
  );
};

const typeNames = () =>
  screen.getAllByRole("checkbox").map((el) => el.textContent ?? "");

describe("SoftwareTypesPicker", () => {
  it("lists every type in the given order", () => {
    renderWithSetup(<ControlledPicker />);

    expect(screen.getByPlaceholderText("Search types")).toBeInTheDocument();
    expect(typeNames()).toEqual(SOFTWARE_TYPES.map((t) => t.displayName));
  });

  it("checks selected types and shows a chip for each", () => {
    renderWithSetup(<ControlledPicker initialKeys={["macos_app"]} />);

    expect(
      screen.getByRole("checkbox", { name: "App (macOS)" })
    ).toHaveAttribute("aria-checked", "true");
    expect(
      screen.getByRole("checkbox", { name: "Browser extension (Brave)" })
    ).toHaveAttribute("aria-checked", "false");
    expect(
      screen.getByRole("button", { name: "App (macOS)" })
    ).toBeInTheDocument();
  });

  it("narrows the list by display name and keeps hidden selections", async () => {
    const { user } = renderWithSetup(<ControlledPicker />);

    await user.click(screen.getByRole("checkbox", { name: "App (macOS)" }));
    await user.type(screen.getByPlaceholderText("Search types"), "EXT");

    await waitFor(() => {
      expect(
        screen.queryByRole("checkbox", { name: "App (macOS)" })
      ).not.toBeInTheDocument();
    });
    const names = typeNames();
    expect(names.length).toBeGreaterThan(0);
    names.forEach((name) => expect(name.toLowerCase()).toContain("ext"));
    expect(
      screen.getByRole("button", { name: "App (macOS)" })
    ).toBeInTheDocument();
  });

  it("says so when no type matches the search", async () => {
    const { user } = renderWithSetup(<ControlledPicker />);

    await user.type(screen.getByPlaceholderText("Search types"), "zzz");

    expect(await screen.findByText("No matching types.")).toBeInTheDocument();
    expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
  });

  it("removes a type when its chip is clicked", async () => {
    const { user } = renderWithSetup(
      <ControlledPicker initialKeys={["macos_app", "brave_extension"]} />
    );

    await user.click(screen.getByRole("button", { name: "App (macOS)" }));

    expect(
      screen.queryByRole("button", { name: "App (macOS)" })
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: "App (macOS)" })
    ).toHaveAttribute("aria-checked", "false");
    expect(
      screen.getByRole("button", { name: "Browser extension (Brave)" })
    ).toBeInTheDocument();
  });
});

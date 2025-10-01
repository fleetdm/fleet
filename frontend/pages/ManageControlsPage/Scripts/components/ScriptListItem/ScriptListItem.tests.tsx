import { render, screen } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";
import { IScript } from "interfaces/script";
import React from "react";
import ScriptListItem from "./ScriptListItem";

const MAC_SCRIPT: IScript = {
  id: 1,
  team_id: 1,
  name: "test_mac_script.sh",
  created_at: "2021-01-01",
  updated_at: "2021-01-01",
};

const WINDOWS_SCRIPT: IScript = {
  id: 1,
  team_id: 1,
  name: "test_win_script.ps1",
  created_at: "2021-01-01",
  updated_at: "2021-01-01",
};

beforeAll(() => {
  jest.useFakeTimers();
  jest.setSystemTime(new Date("2025-05-02T00:00:00Z")); // "over 4 years ago" after created_at
});
afterAll(() => {
  jest.useRealTimers();
});

describe("ScriptListItem", () => {
  const onDelete = jest.fn();
  const onClickScript = jest.fn();
  const onEdit = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("Renders a Script list item with correct platform for macOS", () => {
    render(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    expect(screen.getByText(/macOS & Linux/)).toBeInTheDocument();
  });

  it("Renders a Script list item with correct platform for Windows", () => {
    render(
      <ScriptListItem
        script={WINDOWS_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    expect(screen.getByText(/Windows/)).toBeInTheDocument();
  });

  it("calls onClickScript when script name is clicked", async () => {
    const { user } = renderWithSetup(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    await user.click(screen.getByText("test_mac_script.sh"));
    expect(onClickScript).toHaveBeenCalledWith(MAC_SCRIPT);
  });

  it("only calls onClickScript when clicking elsewhere in the script list item (except 'Edit', see below)", async () => {
    const { user } = renderWithSetup(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    await user.click(screen.getByText("over 4 years ago"));
    expect(onClickScript).toHaveBeenCalledWith(MAC_SCRIPT);
    expect(onEdit).not.toHaveBeenCalled();
    expect(onDelete).not.toHaveBeenCalled();
  });

  it("only calls onDelete when delete button is clicked", async () => {
    const { user } = renderWithSetup(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    await user.click(screen.getByTestId("trash-icon"));
    expect(onDelete).toHaveBeenCalledWith(MAC_SCRIPT);
    expect(onClickScript).not.toHaveBeenCalled();
    expect(onEdit).not.toHaveBeenCalled();
  });

  it("only calls onEdit when pencil button is clicked", async () => {
    const { user } = renderWithSetup(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onEdit={onEdit}
        onClickScript={onClickScript}
      />
    );

    await user.click(screen.getByTestId("pencil-icon"));
    expect(onEdit).toHaveBeenCalledWith(MAC_SCRIPT);
    expect(onClickScript).not.toHaveBeenCalled();
    expect(onDelete).not.toHaveBeenCalled();
  });
});

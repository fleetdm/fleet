import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";
import { IScript } from "interfaces/script";
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

describe("ScriptListItem", () => {
  const onDelete = jest.fn();
  const onClickScript = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("Renders a Script list item with correct platform for macOS", () => {
    render(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
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
        onClickScript={onClickScript}
      />
    );

    expect(screen.getByText(/Windows/)).toBeInTheDocument();
  });

  it("calls onClickScript when script name is clicked", () => {
    render(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onClickScript={onClickScript}
      />
    );

    fireEvent.click(screen.getByText("test_mac_script.sh"));
    expect(onClickScript).toHaveBeenCalledWith(MAC_SCRIPT);
  });

  it("calls onDelete when delete button is clicked", () => {
    render(
      <ScriptListItem
        script={MAC_SCRIPT}
        onDelete={onDelete}
        onClickScript={onClickScript}
      />
    );

    fireEvent.click(screen.getByTestId("trash-icon"));
    expect(onDelete).toHaveBeenCalledWith(MAC_SCRIPT);
  });
});

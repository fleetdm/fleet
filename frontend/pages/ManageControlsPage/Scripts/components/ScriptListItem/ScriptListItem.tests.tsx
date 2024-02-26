import React from "react";

import { render, screen } from "@testing-library/react";

import { IScript } from "interfaces/script";
import ScriptListItem from "./ScriptListItem";

describe("ScriptListItem", () => {
  const onDelete = (script: IScript) => jest.fn();
  it("Renders a Script list item with correct graphic and platform for macOS", () => {
    const script: IScript = {
      id: 1,
      team_id: 1,
      name: "test_mac_script.sh",
      created_at: "2021-01-01",
      updated_at: "2021-01-01",
    };
    render(<ScriptListItem {...{ script, onDelete }} />);

    expect(screen.getByText(/macOS & Linux/)).toBeInTheDocument();
    expect(screen.queryByTestId("file-sh-graphic")).toBeInTheDocument();
  });

  it("Renders a Script list item with correct graphic and platform for Windows", () => {
    const script: IScript = {
      id: 1,
      team_id: 1,
      name: "test_win_script.ps1",
      created_at: "2021-01-01",
      updated_at: "2021-01-01",
    };
    render(<ScriptListItem {...{ script, onDelete }} />);

    expect(screen.getByText(/Windows/)).toBeInTheDocument();
    expect(screen.queryByTestId("file-ps1-graphic")).toBeInTheDocument();
  });
});

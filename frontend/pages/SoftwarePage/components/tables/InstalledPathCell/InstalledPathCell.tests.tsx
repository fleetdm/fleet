import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import { DEFAULT_INSTALLED_VERSION } from "__mocks__/hostMock";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import InstalledPathCell from "./InstalledPathCell";

// Mock subcomponents for isolation
jest.mock(
  "components/TableContainer/DataTable/TextCell",
  () => ({ value, ...props }: any) => (
    <div data-testid="textcell" {...props}>
      {value}
    </div>
  )
);
jest.mock(
  "components/TableContainer/DataTable/TooltipTruncatedTextCell",
  () => ({ value, ...props }: any) => (
    <div data-testid="tooltipcell" {...props}>
      {value}
    </div>
  )
);
jest.mock("components/buttons/Button", () => ({ children, ...props }: any) => (
  <button {...props}>{children}</button>
));

describe("InstalledPathCell", () => {
  it("renders empty cell when installedVersion is null", () => {
    render(
      <InstalledPathCell
        installedVersion={null}
        onClickMultiplePaths={jest.fn()}
      />
    );
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders empty cell when no paths are present", () => {
    render(
      <InstalledPathCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            installed_paths: [],
            signature_information: [
              {
                installed_path: "",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: null,
              },
            ],
          },
        ]}
        onClickMultiplePaths={jest.fn()}
      />
    );
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders single installed path", () => {
    const path = "/Applications/mock.app";
    render(
      <InstalledPathCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            signature_information: [
              {
                installed_path: "/Applications/mock.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: null,
              },
            ],
          },
        ]}
        onClickMultiplePaths={jest.fn()}
      />
    );
    expect(screen.getByTestId("tooltipcell")).toHaveTextContent(path);
  });

  it("renders button for multiple unique paths and calls handler", () => {
    const onClickMultiplePaths = jest.fn();
    render(
      <InstalledPathCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            installed_paths: [
              "/Applications/App1.app",
              "/Applications/App2.app",
              "/Applications/App1.app", // duplicate
            ],
            signature_information: [
              {
                installed_path: "/Applications/mock.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: null,
              },
            ],
          },
        ]}
        onClickMultiplePaths={onClickMultiplePaths}
      />
    );

    // Should show "2 paths" (unique)
    const multiBtn = screen.getByRole("button");
    expect(multiBtn).toHaveTextContent("2 paths");

    fireEvent.click(multiBtn);
    expect(onClickMultiplePaths).toHaveBeenCalled();
  });
});

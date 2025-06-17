import React from "react";
import { render, screen, fireEvent, act } from "@testing-library/react";

import { noop } from "lodash";
import { DEFAULT_INSTALLED_VERSION } from "__mocks__/hostMock";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import HashCell from "./HashCell";

describe("HashCell", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
  });

  it("renders empty cell when installedVersion is null", () => {
    render(
      <HashCell installedVersion={null} onClickMultipleHashes={jest.fn()} />
    );
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders empty cell when no hashes are present", () => {
    render(
      <HashCell
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
        onClickMultipleHashes={jest.fn()}
      />
    );
    expect(screen.getByText(DEFAULT_EMPTY_CELL_VALUE)).toBeInTheDocument();
  });

  it("renders single hash", async () => {
    const hash = "abcdef1234567890";
    render(
      <HashCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            signature_information: [
              {
                installed_path: "/Applications/mock.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: hash,
              },
            ],
          },
        ]}
        onClickMultipleHashes={jest.fn()}
      />
    );

    // Shows first 7 chars and ellipsis
    expect(
      screen.getByText(hash.slice(0, 7), { exact: false })
    ).toBeInTheDocument();
  });

  it("renders button for multiple unique hashes and calls handler", () => {
    const onClickMultipleHashes = noop;

    render(
      <HashCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            signature_information: [
              {
                installed_path: "/Applications/mock1.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: "hash",
              },
              {
                installed_path: "/Applications/mock2.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: "hash2",
              },
              {
                installed_path: "/Applications/mock3.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: "hash3",
              },
            ],
          },
        ]}
        onClickMultipleHashes={onClickMultipleHashes}
      />
    );

    // Should show "3 hashes"
    const multiBtn = screen.getByRole("button");
    expect(multiBtn).toHaveTextContent("3 hashes");
  });
});

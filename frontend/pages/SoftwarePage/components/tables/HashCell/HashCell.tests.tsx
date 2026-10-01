import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { noop } from "lodash";
import React from "react";

import {
  createMockHomebrewInstalledVersion,
  DEFAULT_INSTALLED_VERSION,
} from "__mocks__/hostMock";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

import HashCell from "./HashCell";

describe("HashCell component", () => {
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
                executable_sha256: null,
                executable_path: null,
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
                executable_sha256: null,
                executable_path: null,
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
                executable_sha256: null,
                executable_path: null,
              },
              {
                installed_path: "/Applications/mock2.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: "hash2",
                executable_sha256: null,
                executable_path: null,
              },
              {
                installed_path: "/Applications/mock3.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: "hash3",
                executable_sha256: null,
                executable_path: null,
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

  it("falls back to the executable hash when there is no cdhash", () => {
    const executableHash = "fedcba9876543210";
    render(
      <HashCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            signature_information: [
              {
                installed_path: "/opt/homebrew/Cellar/fortune",
                team_identifier: "",
                hash_sha256: null,
                executable_sha256: executableHash,
                executable_path:
                  "/opt/homebrew/Cellar/fortune/9708/bin/fortune",
              },
            ],
          },
        ]}
        onClickMultipleHashes={jest.fn()}
      />
    );

    expect(
      screen.getByText(executableHash.slice(0, 7), { exact: false })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Copy to clipboard" })
    ).toBeInTheDocument();
  });

  it("renders a link for a Homebrew keg's executables and calls the handler", async () => {
    const user = userEvent.setup();
    const onClickMultipleHashes = jest.fn();

    render(
      <HashCell
        installedVersion={[createMockHomebrewInstalledVersion()]}
        onClickMultipleHashes={onClickMultipleHashes}
      />
    );

    const multiBtn = screen.getByRole("button");
    expect(multiBtn).toHaveTextContent("3 hashes");

    await user.click(multiBtn);
    expect(onClickMultipleHashes).toHaveBeenCalledTimes(1);
  });

  it("prefers the cdhash when an entry carries both hashes", () => {
    const cdHash = "cdhash1234567890";
    render(
      <HashCell
        installedVersion={[
          {
            ...DEFAULT_INSTALLED_VERSION,
            signature_information: [
              {
                installed_path: "/Applications/mock.app",
                team_identifier: "12345TEAMIDENT",
                hash_sha256: cdHash,
                executable_sha256: "executable1234567890",
                executable_path: "/Applications/mock.app/Contents/MacOS/mock",
              },
            ],
          },
        ]}
        onClickMultipleHashes={jest.fn()}
      />
    );

    // One hash, not two, and it is the cdhash.
    expect(screen.queryByRole("button", { name: /hashes/ })).toBeNull();
    expect(
      screen.getByText(cdHash.slice(0, 7), { exact: false })
    ).toBeInTheDocument();
  });

  it("counts hard-linked executables sharing a hash once", () => {
    const sharedHash = "abc1234567890def";
    render(
      <HashCell
        installedVersion={[
          createMockHomebrewInstalledVersion({
            signature_information: [
              {
                installed_path: "/opt/homebrew/Cellar/coreutils",
                team_identifier: "",
                hash_sha256: null,
                executable_sha256: sharedHash,
                executable_path: "/opt/homebrew/Cellar/coreutils/9.5/bin/gln",
              },
              {
                installed_path: "/opt/homebrew/Cellar/coreutils",
                team_identifier: "",
                hash_sha256: null,
                executable_sha256: sharedHash,
                executable_path: "/opt/homebrew/Cellar/coreutils/9.5/bin/glink",
              },
            ],
          }),
        ]}
        onClickMultipleHashes={jest.fn()}
      />
    );

    expect(screen.queryByRole("button", { name: /hashes/ })).toBeNull();
    expect(
      screen.getByText(sharedHash.slice(0, 7), { exact: false })
    ).toBeInTheDocument();
  });
});

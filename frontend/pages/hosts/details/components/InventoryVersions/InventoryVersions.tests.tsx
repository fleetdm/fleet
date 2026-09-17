import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import {
  createMockHomebrewInstalledVersion,
  createMockHostSoftware,
  HOMEBREW_KEG_PATH,
} from "__mocks__/hostMock";

import InventoryVersions from "./InventoryVersions";

const homebrewSoftware = (versions = [createMockHomebrewInstalledVersion()]) =>
  createMockHostSoftware({
    name: "git",
    display_name: "git",
    source: "homebrew_packages",
    bundle_identifier: "",
    installed_versions: versions,
  });

describe("InventoryVersions component", () => {
  beforeAll(() => {
    // jsdom exposes navigator.clipboard as a getter-only property.
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: jest.fn().mockResolvedValue(undefined) },
    });
  });

  it("renders one path and one executable row per Homebrew executable", () => {
    render(<InventoryVersions hostSoftware={homebrewSoftware()} />);

    expect(screen.getAllByText("Path:")).toHaveLength(1);
    expect(screen.getByText(HOMEBREW_KEG_PATH)).toBeInTheDocument();

    expect(screen.getAllByText("Executable:")).toHaveLength(3);
    ["git", "git-cvsserver", "git-upload-pack"].forEach((name) => {
      expect(screen.getByText(name)).toBeInTheDocument();
    });
    // The whole hash is in the markup — the row shortens it with a CSS
    // ellipsis, so copying and text search still get all 64 characters.
    ["aaaaaaa", "bbbbbbb", "ccccccc"].forEach((prefix) => {
      expect(screen.getByText(prefix.padEnd(64, "0"))).toBeInTheDocument();
    });
  });

  it("copies the full executable hash", async () => {
    const user = userEvent.setup();
    const writeText = jest
      .spyOn(navigator.clipboard, "writeText")
      .mockResolvedValue(undefined);

    render(<InventoryVersions hostSoftware={homebrewSoftware()} />);

    const [firstCopy] = screen.getAllByRole("button", {
      name: "Copy to clipboard",
    });
    await user.click(firstCopy);

    expect(writeText).toHaveBeenCalledWith("aaaaaaa".padEnd(64, "0"));
  });

  it("renders the cdhash and no executable row for a signed app", () => {
    render(<InventoryVersions hostSoftware={createMockHostSoftware()} />);

    expect(screen.getByText("Path:")).toBeInTheDocument();
    expect(screen.getByText("/Applications/mock.app")).toBeInTheDocument();
    expect(screen.getByText("Hash:")).toBeInTheDocument();
    expect(screen.getByText("mockhashhere")).toBeInTheDocument();
    expect(screen.queryByText("Executable:")).toBeNull();
  });

  it("renders the path only when an entry has neither hash", () => {
    render(
      <InventoryVersions
        hostSoftware={createMockHostSoftware({
          source: "homebrew_packages",
          installed_versions: [
            createMockHomebrewInstalledVersion({
              signature_information: [
                {
                  installed_path: HOMEBREW_KEG_PATH,
                  team_identifier: "",
                  hash_sha256: null,
                  executable_sha256: null,
                  executable_path: null,
                },
              ],
            }),
          ],
        })}
      />
    );

    expect(screen.getByText("Path:")).toBeInTheDocument();
    expect(screen.queryByText("Hash:")).toBeNull();
    expect(screen.queryByText("Executable:")).toBeNull();
  });

  it("keeps each version's executables to that version", () => {
    const older = createMockHomebrewInstalledVersion({
      version: "2.45.0",
      signature_information: [
        {
          installed_path: HOMEBREW_KEG_PATH,
          team_identifier: "",
          hash_sha256: null,
          executable_sha256: "ddddddd".padEnd(64, "0"),
          executable_path: `${HOMEBREW_KEG_PATH}/2.45.0/bin/git-old`,
        },
      ],
    });

    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([
          older,
          createMockHomebrewInstalledVersion(),
        ])}
      />
    );

    // One executable row for 2.45.0 and three for 2.46.0. Grouping the
    // entries by path alone would repeat every executable in both cards.
    expect(screen.getAllByText("Path:")).toHaveLength(2);
    expect(screen.getAllByText("Executable:")).toHaveLength(4);
    expect(screen.getAllByText("git-old")).toHaveLength(1);
    expect(screen.getAllByText("git-upload-pack")).toHaveLength(1);
  });
});

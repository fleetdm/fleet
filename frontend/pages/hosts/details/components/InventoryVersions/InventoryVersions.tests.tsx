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

/** A keg with `count` executables, each with its own hash, in the shape a
 * formula like netpbm reports. */
const kegWithExecutables = (count: number) =>
  createMockHomebrewInstalledVersion({
    signature_information: Array.from({ length: count }, (_, i) => ({
      installed_path: HOMEBREW_KEG_PATH,
      team_identifier: "",
      hash_sha256: null,
      executable_sha256: `${i}`.padStart(64, "0"),
      executable_path: `${HOMEBREW_KEG_PATH}/2.46.0/bin/tool-${i}`,
    })),
  });

/** Each executable renders one row with its own copy button, which is what
 * distinguishes those rows from the Path and Hash ones. */
const executableRows = () =>
  screen.queryAllByRole("button", { name: "Copy to clipboard" });

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

    expect(executableRows()).toHaveLength(3);
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

    expect(screen.getByText("Hash:")).toBeInTheDocument();
    expect(screen.getByText("mockhashhere")).toBeInTheDocument();
    expect(executableRows()).toHaveLength(0);
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
    expect(executableRows()).toHaveLength(0);
  });

  it("caps a long executables list and labels the rest", () => {
    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([kegWithExecutables(360)])}
      />
    );

    // A formula like netpbm installs hundreds of tools. Mounting a row, a
    // tooltip and a copy button for each one makes the card unusable, so the
    // list stops and says how many are left.
    expect(executableRows()).toHaveLength(10);
    expect(screen.getByText("tool-9")).toBeInTheDocument();
    expect(screen.queryByText("tool-10")).toBeNull();
    expect(screen.getByText("+350 more")).toBeInTheDocument();
  });

  it("does not label a list that fits", () => {
    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([kegWithExecutables(10)])}
      />
    );

    expect(executableRows()).toHaveLength(10);
    expect(screen.queryByText(/more$/)).toBeNull();
  });

  it("copies every hash of the keg, including the ones not shown", async () => {
    const user = userEvent.setup();
    const writeText = jest
      .spyOn(navigator.clipboard, "writeText")
      .mockResolvedValue(undefined);

    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([kegWithExecutables(25)])}
      />
    );

    await user.click(screen.getByRole("button", { name: "Copy all hashes" }));

    const expected = Array.from({ length: 25 }, (_, i) =>
      `${i}`.padStart(64, "0")
    ).join("\n");
    expect(writeText).toHaveBeenCalledWith(expected);
  });

  it("copies hard-linked executables sharing a hash once", async () => {
    const user = userEvent.setup();
    const writeText = jest
      .spyOn(navigator.clipboard, "writeText")
      .mockResolvedValue(undefined);
    const sharedHash = "abc1234".padEnd(64, "0");

    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([
          createMockHomebrewInstalledVersion({
            signature_information: ["gln", "glink"].map((binary) => ({
              installed_path: HOMEBREW_KEG_PATH,
              team_identifier: "",
              hash_sha256: null,
              executable_sha256: sharedHash,
              executable_path: `${HOMEBREW_KEG_PATH}/2.46.0/bin/${binary}`,
            })),
          }),
        ])}
      />
    );

    await user.click(screen.getByRole("button", { name: "Copy all hashes" }));
    expect(writeText).toHaveBeenCalledWith(sharedHash);
  });

  it("offers no copy-all for a single executable", () => {
    render(
      <InventoryVersions
        hostSoftware={homebrewSoftware([kegWithExecutables(1)])}
      />
    );

    expect(
      screen.queryByRole("button", { name: "Copy all hashes" })
    ).toBeNull();
    // The row's own copy button still covers the one hash.
    expect(
      screen.getByRole("button", { name: "Copy to clipboard" })
    ).toBeInTheDocument();
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
    expect(executableRows()).toHaveLength(4);
    expect(screen.getAllByText("git-old")).toHaveLength(1);
    expect(screen.getAllByText("git-upload-pack")).toHaveLength(1);
  });
});

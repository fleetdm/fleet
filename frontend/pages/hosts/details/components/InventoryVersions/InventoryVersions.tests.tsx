import React from "react";
import { noop } from "lodash";
import { render, screen } from "@testing-library/react";

import { createMockHostSoftware } from "__mocks__/hostMock";

import InventoryVersions from "./InventoryVersions";

describe("InventoryVersions", () => {
  it("appends the Go toolchain version for a go_binaries row", () => {
    render(
      <InventoryVersions
        hostSoftware={createMockHostSoftware({
          source: "go_binaries",
          installed_versions: [
            {
              version: "v0.21.1",
              release: "go1.26.1",
              bundle_identifier: "",
              vulnerabilities: null,
              installed_paths: [],
            },
          ],
        })}
      />
    );

    expect(screen.getByText("v0.21.1 (go1.26.1)")).toBeInTheDocument();
  });

  // The same binary version built with two toolchains is two installed
  // versions with the same version string.
  it("renders a card per installed version when two share a version string", () => {
    const consoleError = jest.spyOn(console, "error").mockImplementation(noop);

    render(
      <InventoryVersions
        hostSoftware={createMockHostSoftware({
          source: "go_binaries",
          installed_versions: [
            {
              version: "v0.21.1",
              release: "go1.26.1",
              bundle_identifier: "",
              vulnerabilities: null,
              installed_paths: [],
            },
            {
              version: "v0.21.1",
              release: "go1.25.4",
              bundle_identifier: "",
              vulnerabilities: null,
              installed_paths: [],
            },
          ],
        })}
      />
    );

    expect(screen.getByText("v0.21.1 (go1.26.1)")).toBeInTheDocument();
    expect(screen.getByText("v0.21.1 (go1.25.4)")).toBeInTheDocument();
    const duplicateKeyWarnings = consoleError.mock.calls.filter((call) =>
      String(call[0]).includes("same key")
    );
    expect(duplicateKeyWarnings).toEqual([]);

    consoleError.mockRestore();
  });
});

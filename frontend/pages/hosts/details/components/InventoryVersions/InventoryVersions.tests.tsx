import React from "react";
import { render, screen } from "@testing-library/react";

import { createMockHostSoftware } from "__mocks__/hostMock";

import InventoryVersions from "./InventoryVersions";

describe("InventoryVersions", () => {
  // This card builds the formatSoftwareVersion argument field by field rather
  // than passing a whole row, so dropping `release` would compile and silently
  // stop showing the toolchain version. installed_versions[] entries carry no
  // source of their own, so the card supplies the host software row's.
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
});

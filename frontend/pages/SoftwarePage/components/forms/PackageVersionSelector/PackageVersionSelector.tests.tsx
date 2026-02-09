import React from "react";
import { render, screen } from "@testing-library/react";
import { noop } from "lodash";

import PackageVersionSelector from "./PackageVersionSelector";

describe("PackageVersionSelector component", () => {
  it("renders the package version dropdown when there are package versions to choose from", () => {
    render(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versions={[
          { value: "1.0.0", label: "1.0.0" },
          { value: "2.0.0", label: "2.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    expect(screen.getByRole("option", { name: "1.0.0" })).toBeVisible();
    expect(screen.getByRole("option", { name: "2.0.0" })).toBeVisible();
  });
});

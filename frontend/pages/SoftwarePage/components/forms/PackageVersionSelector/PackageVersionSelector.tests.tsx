import React from "react";
import { noop } from "lodash";
import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import PackageVersionSelector from "./PackageVersionSelector";

describe("PackageVersionSelector component", () => {
  it("returns null when there are no version options", () => {
    const { container } = render(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[]}
        onSelectVersion={noop}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it("renders the package version dropdown when there are package versions to choose from", () => {
    render(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[
          { value: "1.0.0", label: "1.0.0" },
          { value: "2.0.0", label: "2.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    expect(screen.getByText("1.0.0")).toBeInTheDocument();
  });

  it("disables the dropdown when the selected version is the first option", () => {
    render(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[
          { value: "1.0.0", label: "1.0.0" },
          { value: "2.0.0", label: "2.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    const combobox = screen.getByRole("combobox");
    expect(combobox).toBeDisabled();
  });

  it("shows the GitOps rollback tooltip text when the selected version is the first option", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[
          { value: "1.0.0", label: "1.0.0" },
          { value: "2.0.0", label: "2.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    // TooltipWrapper is wrapping the select; it attaches tooltip to this element:
    const tooltipAnchor = document.querySelector(
      ".component__tooltip-wrapper__element"
    ) as HTMLElement;

    await user.hover(tooltipAnchor);

    await waitFor(() => {
      expect(
        screen.getByText("Currently, you can only use GitOps", { exact: false })
      ).toBeInTheDocument();
      expect(
        screen.getByText("to roll back (UI coming soon).", { exact: false })
      ).toBeInTheDocument();
    });
  });

  it("shows the update-to-latest tooltip text when the selected version is not the first option", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="2.0.0"
        versionOptions={[
          { value: "1.0.0", label: "1.0.0" },
          { value: "2.0.0", label: "2.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    const tooltipAnchor = document.querySelector(
      ".component__tooltip-wrapper__element"
    ) as HTMLElement;

    await user.hover(tooltipAnchor);

    await waitFor(() => {
      expect(
        screen.getByText("Currently, to update to latest you have", {
          exact: false,
        })
      ).toBeInTheDocument();
      expect(
        screen.getByText("to delete and re-add the software.", { exact: false })
      ).toBeInTheDocument();
    });
  });
});

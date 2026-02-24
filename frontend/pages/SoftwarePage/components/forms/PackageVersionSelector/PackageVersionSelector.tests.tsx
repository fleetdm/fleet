import React from "react";
import { noop } from "lodash";
import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import PackageVersionSelector from "./PackageVersionSelector";

describe("PackageVersionSelector component", () => {
  it("returns null when there are no version options", () => {
    const { container } = render(
      <PackageVersionSelector
        selectedVersion="2.0.0"
        versionOptions={[]}
        onSelectVersion={noop}
      />
    );

    expect(container.firstChild).toBeNull();
  });

  it("renders the package version dropdown when there are package versions to choose from", () => {
    render(
      <PackageVersionSelector
        selectedVersion="2.0.0"
        versionOptions={[
          { value: "2.0.0", label: "Latest (2.0.0)" },
          { value: "1.0.0", label: "1.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    // Renders the label for the selected (latest) version
    expect(screen.getByText("Latest (2.0.0)")).toBeInTheDocument();
  });

  it("disables all non-selected options when the latest version is selected", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="2.0.0"
        versionOptions={[
          { value: "2.0.0", label: "Latest (2.0.0)" }, // selected
          { value: "1.0.0", label: "1.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    const combobox = screen.getByRole("combobox");
    await user.click(combobox);

    const optionInnerDivs = screen.getAllByTestId("dropdown-option");

    const latestInner = optionInnerDivs.find(
      (el) => el.textContent === "Latest (2.0.0)"
    );
    const oldInner = optionInnerDivs.find((el) => el.textContent === "1.0.0");

    expect(latestInner).toBeDefined();
    expect(oldInner).toBeDefined();

    const latestOptionWrapper = latestInner?.closest(
      ".react-select__option"
    ) as HTMLElement | null;
    const oldOptionWrapper = oldInner?.closest(
      ".react-select__option"
    ) as HTMLElement | null;

    expect(latestOptionWrapper).not.toBeNull();
    expect(oldOptionWrapper).not.toBeNull();

    // Selected option (Latest 2.0.0) is enabled
    expect(latestOptionWrapper).toHaveAttribute("aria-disabled", "false");
    // Non-selected option (1.0.0) is disabled
    expect(oldOptionWrapper).toHaveAttribute("aria-disabled", "true");
  });

  it("disables all non-selected options when a non-latest version is selected", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[
          { value: "2.0.0", label: "Latest (2.0.0)" },
          { value: "1.0.0", label: "1.0.0" }, // selected
        ]}
        onSelectVersion={noop}
      />
    );

    const combobox = screen.getByRole("combobox");
    await user.click(combobox);

    const optionInnerDivs = screen.getAllByTestId("dropdown-option");

    const latestInner = optionInnerDivs.find(
      (el) => el.textContent === "Latest (2.0.0)"
    );
    const oldInner = optionInnerDivs.find((el) => el.textContent === "1.0.0");

    expect(latestInner).toBeDefined();
    expect(oldInner).toBeDefined();

    const latestOptionWrapper = latestInner?.closest(
      ".react-select__option"
    ) as HTMLElement | null;
    const oldOptionWrapper = oldInner?.closest(
      ".react-select__option"
    ) as HTMLElement | null;

    expect(latestOptionWrapper).not.toBeNull();
    expect(oldOptionWrapper).not.toBeNull();

    // Selected option (1.0.0) is enabled
    expect(oldOptionWrapper).toHaveAttribute("aria-disabled", "false");
    // Non-selected option (Latest 2.0.0) is disabled
    expect(latestOptionWrapper).toHaveAttribute("aria-disabled", "true");
  });

  it("shows the GitOps rollback tooltip text when the selected version is the first (latest) option", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="2.0.0"
        versionOptions={[
          { value: "2.0.0", label: "Latest (2.0.0)" }, // first / latest
          { value: "1.0.0", label: "1.0.0" },
        ]}
        onSelectVersion={noop}
      />
    );

    // TooltipWrapper attaches tooltip to this element:
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

  it("shows the update-to-latest tooltip text when the selected version is not the first (latest) option", async () => {
    const { user } = renderWithSetup(
      <PackageVersionSelector
        selectedVersion="1.0.0"
        versionOptions={[
          { value: "2.0.0", label: "Latest (2.0.0)" }, // first / latest
          { value: "1.0.0", label: "1.0.0" },
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

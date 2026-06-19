import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";

import SoftwareFilters from "./SoftwareFilters";

const emptyVulnsHandler = http.get(baseUrl("/vulnerabilities"), () =>
  HttpResponse.json({
    count: 0,
    counts_updated_at: "",
    vulnerabilities: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const baseProps = {
  categories: [...ALL_CVE_SOFTWARE_CATEGORY_VALUES],
  knownExploit: false,
  epssMin: "",
  epssMax: "",
  excludeCVEs: [],
  setCategories: jest.fn(),
  setKnownExploit: jest.fn(),
  setEpssMin: jest.fn(),
  setEpssMax: jest.fn(),
  setExcludeCVEs: jest.fn(),
};

const render = createCustomRenderer({ withBackendMock: true });

describe("SoftwareFilters", () => {
  beforeEach(() => mockServer.use(emptyVulnsHandler));

  it("shows all software categories checked by default and KEV unchecked", () => {
    render(<SoftwareFilters {...baseProps} />);

    // The visible checkbox is a div[role=checkbox] whose accessible name is the
    // `name` prop (e.g. "category-os"); its checked state is aria-checked.
    expect(screen.getByRole("checkbox", { name: "category-os" })).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "category-browsers" })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "category-office" })
    ).toBeChecked();
    expect(
      screen.getByRole("checkbox", { name: "category-adobe" })
    ).toBeChecked();

    // Human-readable labels are rendered too.
    expect(screen.getByText("Operating system (OS)")).toBeInTheDocument();

    expect(
      screen.getByRole("checkbox", { name: "known-exploit" })
    ).not.toBeChecked();
  });

  it("removes a category from the set when unchecked", async () => {
    const setCategories = jest.fn();
    const { user } = render(
      <SoftwareFilters {...baseProps} setCategories={setCategories} />
    );

    await user.click(
      screen.getByRole("checkbox", { name: "category-browsers" })
    );

    expect(setCategories).toHaveBeenCalledWith(
      ALL_CVE_SOFTWARE_CATEGORY_VALUES.filter((c) => c !== "browsers")
    );
  });

  it("shows a validation error when no category is selected", () => {
    render(<SoftwareFilters {...baseProps} categories={[]} />);

    expect(
      screen.getByText("Select at least one software category.")
    ).toBeInTheDocument();
  });

  it("does not show the category error when a category is selected", () => {
    render(<SoftwareFilters {...baseProps} categories={["os"]} />);

    expect(
      screen.queryByText("Select at least one software category.")
    ).not.toBeInTheDocument();
  });

  it("keeps Advanced options collapsed until toggled", async () => {
    const { user } = render(<SoftwareFilters {...baseProps} />);

    expect(
      screen.queryByText("Probability of exploit")
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Advanced options/i }));

    expect(screen.getByText("Probability of exploit")).toBeInTheDocument();
    expect(
      screen.getByText("Exclude vulnerabilities (CVEs)")
    ).toBeInTheDocument();
  });

  it("surfaces an EPSS range error for out-of-range input", async () => {
    const { user } = render(<SoftwareFilters {...baseProps} epssMin="-1" />);

    await user.click(screen.getByRole("button", { name: /Advanced options/i }));

    expect(screen.getByText("Must be from 0 to 100")).toBeInTheDocument();
  });

  it("renders excluded CVEs as removable pills", async () => {
    const setExcludeCVEs = jest.fn();
    const { user } = render(
      <SoftwareFilters
        {...baseProps}
        excludeCVEs={["CVE-2025-0001"]}
        setExcludeCVEs={setExcludeCVEs}
      />
    );

    await user.click(screen.getByRole("button", { name: /Advanced options/i }));

    const pill = screen.getByRole("button", { name: /CVE-2025-0001/i });
    expect(pill).toBeInTheDocument();

    await user.click(pill);
    expect(setExcludeCVEs).toHaveBeenCalledWith([]);
  });
});

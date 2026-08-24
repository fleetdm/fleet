import React from "react";
import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { createCustomRenderer, baseUrl } from "test/test-utils";
import mockServer from "test/mock-server";
import { ALL_CVE_SOFTWARE_CATEGORY_VALUES } from "interfaces/charts";
import { SEVERITY_SCORE_RANGE_ERROR } from "components/SeverityFilter";

import SoftwareFilters from "./SoftwareFilters";
import { EPSS_RANGE_HELP, NO_CATEGORIES_MSG } from "./helpers";

const emptyVulnsHandler = http.get(baseUrl("/vulnerabilities"), () =>
  HttpResponse.json({
    count: 0,
    counts_updated_at: "",
    vulnerabilities: [],
    meta: { has_next_results: false, has_previous_results: false },
  })
);

const baseProps: React.ComponentProps<typeof SoftwareFilters> = {
  categories: [...ALL_CVE_SOFTWARE_CATEGORY_VALUES],
  knownExploit: false,
  epssMin: "",
  epssMax: "",
  severityFilter: { severity: "critical", minScore: "9", maxScore: "10" },
  errors: {},
  excludeCVEs: [],
  setCategories: jest.fn(),
  setKnownExploit: jest.fn(),
  setEpssMin: jest.fn(),
  setEpssMax: jest.fn(),
  setSeverityFilter: jest.fn(),
  setExcludeCVEs: jest.fn(),
  onFieldBlur: jest.fn(),
  onFieldFocus: jest.fn(),
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

  // Errors are owned by the parent, which decides when a field has earned one.
  it("renders the category error the parent is showing", () => {
    render(
      <SoftwareFilters
        {...baseProps}
        categories={[]}
        errors={{ categories: NO_CATEGORIES_MSG }}
      />
    );

    expect(screen.getByText(NO_CATEGORIES_MSG)).toBeInTheDocument();
  });

  it("shows no category error while the parent is reporting none", () => {
    render(<SoftwareFilters {...baseProps} categories={[]} />);

    expect(screen.queryByText(NO_CATEGORIES_MSG)).not.toBeInTheDocument();
  });

  it("keeps Advanced options collapsed until toggled", async () => {
    const { user } = render(<SoftwareFilters {...baseProps} />);

    expect(
      screen.queryByText("Probability of exploit")
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Advanced options/i }));

    expect(screen.getByText("Probability of exploit")).toBeInTheDocument();
    expect(screen.getByText("Severity")).toBeInTheDocument();
    expect(
      screen.getByText("Exclude vulnerabilities (CVEs)")
    ).toBeInTheDocument();

    // And back, since nothing is holding it open.
    await user.click(screen.getByRole("button", { name: /Advanced options/i }));
    expect(
      screen.queryByText("Probability of exploit")
    ).not.toBeInTheDocument();
  });

  it("opens Advanced options on mount when asked, and still collapses", async () => {
    const { user } = render(
      <SoftwareFilters {...baseProps} initialShowAdvanced />
    );

    expect(screen.getByText("Probability of exploit")).toBeInTheDocument();

    // Only the starting state, not a lock — nothing is holding it open.
    await user.click(screen.getByRole("button", { name: /Advanced options/i }));
    expect(
      screen.queryByText("Probability of exploit")
    ).not.toBeInTheDocument();
  });

  // A collapsed reveal would hide the only thing standing between the user and
  // a successful Apply.
  it("force-opens Advanced options over an error and refuses to collapse", async () => {
    const { user } = render(
      <SoftwareFilters
        {...baseProps}
        epssMin="-1"
        severityFilter={{ severity: "custom", minScore: "11", maxScore: "" }}
        errors={{
          epssMin: EPSS_RANGE_HELP,
          cvssMin: SEVERITY_SCORE_RANGE_ERROR,
        }}
      />
    );

    // Open on mount, with no click. Matched by message, since FormField renders
    // an error in the label's place.
    expect(screen.getByText(EPSS_RANGE_HELP)).toBeInTheDocument();
    expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /Advanced options/i }));

    expect(screen.getByText(EPSS_RANGE_HELP)).toBeInTheDocument();
    expect(screen.getByText(SEVERITY_SCORE_RANGE_ERROR)).toBeInTheDocument();
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

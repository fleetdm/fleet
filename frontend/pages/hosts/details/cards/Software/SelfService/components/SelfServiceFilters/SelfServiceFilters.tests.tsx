import React from "react";
import { screen } from "@testing-library/react";

import { createCustomRenderer } from "test/test-utils";
import { createMockSelfServiceCategory } from "test/handlers/self-service-categories-handlers";

import SelfServiceFilters from "./SelfServiceFilters";

const baseProps = {
  query: "",
  categoryId: undefined,
  categories: [],
  onSearchQueryChange: jest.fn(),
  onCategoryChange: jest.fn(),
};

describe("SelfServiceFilters", () => {
  it("renders the search field", () => {
    const render = createCustomRenderer();
    render(<SelfServiceFilters {...baseProps} />);

    expect(screen.getByPlaceholderText("Search by name")).toBeInTheDocument();
  });

  it("hides the CategoryFilter when the categories list is empty", () => {
    const render = createCustomRenderer();
    render(<SelfServiceFilters {...baseProps} categories={[]} />);

    // The CategoryFilter trigger is a button with aria-expanded; without the
    // CategoryFilter it shouldn't appear at all.
    expect(
      screen.queryByRole("button", { expanded: false })
    ).not.toBeInTheDocument();
  });

  it("renders the CategoryFilter when categories are present", () => {
    const render = createCustomRenderer();
    render(
      <SelfServiceFilters
        {...baseProps}
        categories={[
          createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
        ]}
      />
    );

    expect(screen.getByRole("button", { expanded: false })).toBeInTheDocument();
  });

  it("renders the install-all slot when provided", () => {
    const render = createCustomRenderer();
    render(
      <SelfServiceFilters
        {...baseProps}
        installAllSlot={<button type="button">Slot button</button>}
      />
    );

    expect(
      screen.getByRole("button", { name: /Slot button/i })
    ).toBeInTheDocument();
  });

  it("does not render an install-all slot when omitted (mobile path)", () => {
    const render = createCustomRenderer();
    render(<SelfServiceFilters {...baseProps} />);

    // Only the search field's elements should be in the actions area.
    expect(
      screen.queryByRole("button", { name: /Slot button/i })
    ).not.toBeInTheDocument();
  });
});

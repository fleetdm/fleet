import React from "react";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { createCustomRenderer } from "test/test-utils";
import { createMockSelfServiceCategory } from "test/handlers/self-service-categories-handlers";

import CategoryFilter from "./CategoryFilter";

const CATEGORIES = [
  createMockSelfServiceCategory({ id: 1, name: "🌎 Browsers" }),
  createMockSelfServiceCategory({ id: 2, name: "🔐 Security" }),
];

// react-select-5 doesn't apply role="listbox" to the menu; scope queries by
// the menu's CSS class instead to avoid matching the trigger's own label.
const getMenu = () => {
  const menu = document.querySelector(".self-service-category-filter__menu");
  if (!menu) throw new Error("category filter menu is not open");
  return menu as HTMLElement;
};

const openMenu = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(screen.getByRole("button"));
  await screen.findByPlaceholderText("Search categories");
};

describe("CategoryFilter", () => {
  it("renders 'All' as the selected label when no category is selected", () => {
    const render = createCustomRenderer();
    render(<CategoryFilter categories={CATEGORIES} onChange={jest.fn()} />);

    expect(screen.getByRole("button")).toHaveTextContent("All");
  });

  it("renders the matching category name when one is selected", () => {
    const render = createCustomRenderer();
    render(
      <CategoryFilter
        categories={CATEGORIES}
        selectedCategoryId={1}
        onChange={jest.fn()}
      />
    );

    expect(screen.getByRole("button")).toHaveTextContent("🌎 Browsers");
  });

  it("opens the menu and shows all category options when the trigger is clicked", async () => {
    const render = createCustomRenderer();
    const user = userEvent.setup();
    render(
      <CategoryFilter
        categories={CATEGORIES}
        selectedCategoryId={1}
        onChange={jest.fn()}
      />
    );

    await openMenu(user);

    expect(within(getMenu()).getByText("All")).toBeInTheDocument();
    expect(within(getMenu()).getByText("🔐 Security")).toBeInTheDocument();
    expect(within(getMenu()).getByText("🌎 Browsers")).toBeInTheDocument();
  });

  it("filters options by search query, including 'All'", async () => {
    const render = createCustomRenderer();
    const user = userEvent.setup();
    render(
      <CategoryFilter
        categories={CATEGORIES}
        selectedCategoryId={1}
        onChange={jest.fn()}
      />
    );

    await openMenu(user);
    await user.type(screen.getByPlaceholderText("Search categories"), "secur");

    expect(within(getMenu()).getByText("🔐 Security")).toBeInTheDocument();
    expect(
      within(getMenu()).queryByText("🌎 Browsers")
    ).not.toBeInTheDocument();
    expect(within(getMenu()).queryByText("All")).not.toBeInTheDocument();
  });

  it("shows the no-match message when the search filters everything out", async () => {
    const render = createCustomRenderer();
    const user = userEvent.setup();
    render(
      <CategoryFilter
        categories={CATEGORIES}
        selectedCategoryId={1}
        onChange={jest.fn()}
      />
    );

    await openMenu(user);
    await user.type(
      screen.getByPlaceholderText("Search categories"),
      "zzznoexist"
    );

    expect(
      screen.getByText("No categories match this search.")
    ).toBeInTheDocument();
  });

  it("calls onChange with the category id when an option is picked", async () => {
    const onChange = jest.fn();
    const render = createCustomRenderer();
    const user = userEvent.setup();
    render(<CategoryFilter categories={CATEGORIES} onChange={onChange} />);

    await openMenu(user);
    await user.click(within(getMenu()).getByText("🔐 Security"));

    expect(onChange).toHaveBeenCalledWith(2);
  });

  it("calls onChange with undefined when 'All' is picked", async () => {
    const onChange = jest.fn();
    const render = createCustomRenderer();
    const user = userEvent.setup();
    render(
      <CategoryFilter
        categories={CATEGORIES}
        selectedCategoryId={1}
        onChange={onChange}
      />
    );

    await openMenu(user);
    await user.click(within(getMenu()).getByText("All"));

    expect(onChange).toHaveBeenCalledWith(undefined);
  });

  it("disables the trigger when isDisabled is true", () => {
    const render = createCustomRenderer();
    render(
      <CategoryFilter categories={CATEGORIES} onChange={jest.fn()} isDisabled />
    );

    expect(screen.getByRole("button")).toBeDisabled();
  });
});

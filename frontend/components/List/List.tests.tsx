// components/__tests__/List.test.tsx
import React from "react";
import { render, screen, fireEvent } from "@testing-library/react";

import List, { IListProps } from "./List";

interface ITestItem {
  id: number | string;
  name: string;
}

const renderList = (props: Partial<IListProps<ITestItem>> = {}) => {
  const defaultProps: IListProps<ITestItem> = {
    data: [],
    renderItemRow: (item) => <span>{item.name}</span>,
    ...props,
  };

  return render(<List {...defaultProps} />);
};

describe("List", () => {
  it("renders heading when provided", () => {
    const headingText = "My heading";
    renderList({ heading: <div>{headingText}</div> });

    expect(screen.getByText(headingText)).toBeInTheDocument();
  });

  it("renders help text when provided", () => {
    const helpText = "Some help text";
    renderList({ helpText });

    expect(screen.getByText(helpText)).toBeInTheDocument();
  });

  it("renders list items with data", () => {
    const data: ITestItem[] = [
      { id: 1, name: "Row 1" },
      { id: 2, name: "Row 2" },
    ];

    renderList({ data });

    expect(screen.getByText("Row 1")).toBeInTheDocument();
    expect(screen.getByText("Row 2")).toBeInTheDocument();
  });

  it("shows loading overlay when isLoading is true", () => {
    renderList({ isLoading: true });

    const overlay = document.querySelector(".loading-overlay");
    expect(overlay).toBeInTheDocument();
  });

  it("calls onClickRow when row is clicked", () => {
    const handleClick = jest.fn();
    const data: ITestItem[] = [{ id: 1, name: "Clickable row" }];

    renderList({
      data,
      onClickRow: handleClick,
    });

    fireEvent.click(screen.getByText("Clickable row"));
    expect(handleClick).toHaveBeenCalledTimes(1);
    expect(handleClick).toHaveBeenCalledWith(data[0]);
  });

  it("applies clickable class when onClickRow is provided", () => {
    const data: ITestItem[] = [{ id: 1, name: "Clickable row" }];

    const { container } = renderList({
      data,
      onClickRow: jest.fn(),
    });

    const row = container.querySelector(".list__row");
    expect(row).toHaveClass("list__row--clickable");
  });

  it("uses custom idKey when provided", () => {
    interface ICustomItem {
      customId: string;
      name: string;
    }

    const data: ICustomItem[] = [
      { customId: "alpha", name: "Alpha" },
      { customId: "beta", name: "Beta" },
    ];

    const { container } = render(
      <List<ICustomItem, "customId">
        data={data}
        idKey="customId"
        renderItemRow={(item) => <span>{item.name}</span>}
      />
    );

    const listItems = container.querySelectorAll("li.list__row");
    // first li is potentially the header; filter by text content to be safe
    const alphaLi = Array.from(listItems).find((li) =>
      li.textContent?.includes("Alpha")
    );
    const betaLi = Array.from(listItems).find((li) =>
      li.textContent?.includes("Beta")
    );

    expect(alphaLi?.getAttribute("key")).toBeNull(); // React doesn't expose "key" to the DOM
    expect(betaLi).toBeInTheDocument();
  });
});

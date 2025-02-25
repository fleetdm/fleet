import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import PaginatedList from "./PaginatedList";
import { k } from "msw/lib/core/GraphQLHandler-D6mLMXGZ";

describe("PaginatedList", () => {
  interface ITestItem {
    id: number;
    name: string;
    key: string;
    val: string;
    favoriteIceCreamFlavor: string;
  }

  const smallSet = [
    {
      id: 1,
      name: "Item 1",
      favoriteIceCreamFlavor: "Vanilla",
      key: "UNO",
      val: "ONE",
    },
    {
      id: 2,
      name: "Item 2",
      favoriteIceCreamFlavor: "Dirt",
      key: "DOS",
      val: "TWO",
    },
    {
      id: 3,
      name: "Item 3",
      favoriteIceCreamFlavor: "Sadness",
      key: "TRES",
      val: "THREE",
    },
  ];
  const fetchSmallSet = (pageNumber: number) => {
    return Promise.resolve(smallSet);
  };

  const fetchLargetSet = (pageNumber: number) => {};

  it("Renders a list of items with checkboxes", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchSmallSet}
        idKey="id"
        labelKey="name"
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitFor(() => {
      expect(
        container.querySelector(".loading-overlay")
      ).not.toBeInTheDocument();
    });
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    smallSet.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
    });
  });

  it("Supports custom id and label properties", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchSmallSet}
        idKey="key"
        labelKey="val"
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitFor(() => {
      expect(
        container.querySelector(".loading-overlay")
      ).not.toBeInTheDocument();
    });
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    smallSet.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.val);
      expect(checkboxes[index]).toHaveAccessibleName(
        `item_${item.key}_checkbox`
      );
    });
  });

  it("Supports setting selected items based on a property", async () => {});
  it("Supports setting selected items based on a function", async () => {});
  it("Adds pagination when > page size items are returned", async () => {});
  it("Allows for custom markup in item rows", async () => {});
  it("Allows for custom markup for item labels", async () => {});
  it("Notifies the parent when an item is toggled", async () => {});
  it("Notifies the parent when a change is made using custom markup", async () => {});
  it("Notifies the parent when a change is made to the set of dirty items", async () => {});
  it("Exposes a method to get the set of changed items", async () => {});
  it("Allows for disabling the list", async () => {});
  it("Allows for disabling the save button with a custom message", async () => {});
});

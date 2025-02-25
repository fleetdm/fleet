import React from "react";
import { screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

import PaginatedList from "./PaginatedList";

describe("PaginatedList", () => {
  interface ITestItem {
    id: number;
    name: string;
    key: string;
    val: string;
    favoriteIceCreamFlavor: string;
    checkMeBruh: boolean;
  }

  const items = [
    {
      id: 1,
      name: "Item 1",
      favoriteIceCreamFlavor: "Vanilla",
      key: "UNO",
      val: "ONE",
      checkMeBruh: true,
    },
    {
      id: 2,
      name: "Item 2",
      favoriteIceCreamFlavor: "Dirt",
      key: "DOS",
      val: "TWO",
      checkMeBruh: false,
    },
    {
      id: 3,
      name: "Item 3",
      favoriteIceCreamFlavor: "Sadness",
      key: "TRES",
      val: "THREE",
      checkMeBruh: false,
    },
  ];

  const fetchSmallPage = (pageNumber: number) => {
    if (pageNumber === 0) {
      return Promise.resolve([items[0], items[1]]);
    } else if (pageNumber === 1) {
      return Promise.resolve([items[2]]);
    }
    throw new Error("Invalid page number");
  };

  const fetchLargePage = (pageNumber: number) => {
    if (pageNumber === 0) {
      return Promise.resolve(items);
    }
    throw new Error("Invalid page number");
  };

  const checkPaginationIsDisabled = () => {
    const nextButton = screen.getByRole("button", { name: /next/i });
    const previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeDisabled();
    expect(previousButton).toBeDisabled();
  };

  const waitForLoadingToFinish = async (container: HTMLElement) => {
    await waitFor(() => {
      expect(
        container.querySelector(".loading-overlay")
      ).not.toBeInTheDocument();
    });
  };

  it("Renders a list of items with checkboxes", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });
    checkPaginationIsDisabled();
  });

  it("Supports custom id and label properties", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        idKey="key"
        labelKey="val"
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.val);
      expect(checkboxes[index]).toHaveAccessibleName(
        `item_${item.key}_checkbox`
      );
      expect(checkboxes[index]).not.toBeChecked();
    });
    checkPaginationIsDisabled();
  });

  it("Supports setting selected items based on a property", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected="checkMeBruh"
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      if (item.checkMeBruh) {
        expect(checkboxes[index]).toBeChecked();
      } else {
        expect(checkboxes[index]).not.toBeChecked();
      }
    });
    checkPaginationIsDisabled();
  });

  it("Supports setting selected items based on a function", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={(item) => item.favoriteIceCreamFlavor === "Dirt"}
      />
    );
    await waitForLoadingToFinish(container);

    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      if (item.favoriteIceCreamFlavor === "Dirt") {
        expect(checkboxes[index]).toBeChecked();
      } else {
        expect(checkboxes[index]).not.toBeChecked();
      }
    });
    checkPaginationIsDisabled();
  });

  it("Adds pagination when > page size items are returned", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchSmallPage}
        pageSize={2}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);

    let checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    [items[0], items[1]].forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });

    let nextButton = screen.getByRole("button", { name: /next/i });
    let previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeEnabled();
    expect(previousButton).toBeDisabled();

    await waitFor(() => {
      nextButton.click();
    });

    await waitForLoadingToFinish(container);

    checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(checkboxes[0]).toHaveTextContent(items[2].name);
    expect(checkboxes[0]).not.toBeChecked();
    nextButton = screen.getByRole("button", { name: /next/i });
    previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeDisabled();
    expect(previousButton).toBeEnabled();
  });

  it("Allows for custom markup in item rows", async () => {});

  it("Allows for custom markup for item labels", async () => {});

  it("Notifies the parent when an item is toggled", async () => {});

  it("Notifies the parent when a change is made using custom markup", async () => {});

  it("Notifies the parent when a change is made to the set of dirty items", async () => {});

  it("Exposes a method to get the set of changed items", async () => {});

  it("Allows for disabling the list", async () => {});

  it("Allows for disabling the save button with a custom message", async () => {});
});

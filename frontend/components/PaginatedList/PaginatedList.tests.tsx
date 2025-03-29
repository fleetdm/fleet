import React, { createRef } from "react";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithSetup } from "test/test-utils";

import PaginatedList, { IPaginatedListHandle } from "./PaginatedList";

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

  const fetchTinyPage = (pageNumber: number) => {
    if (pageNumber <= 2) {
      return Promise.resolve([items[pageNumber]]);
    }
    throw new Error("Invalid page number");
  };

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

  const checkPaginationIsHidden = () => {
    const nextButton = screen.queryByText(/next/i);
    const previousButton = screen.queryByText(/previous/i);
    expect(nextButton).not.toBeInTheDocument();
    expect(previousButton).not.toBeInTheDocument();
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
    checkPaginationIsHidden();
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
    checkPaginationIsHidden();
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
    checkPaginationIsHidden();
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
    checkPaginationIsHidden();
  });

  it("Adds pagination when > page size items are returned (without fetchCount provided)", async () => {
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

    // Check the first page.
    let checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    [items[0], items[1]].forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });

    // Move to second page.
    let nextButton = screen.getByRole("button", { name: /next/i });
    let previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeEnabled();
    expect(previousButton).toBeDisabled();

    await userEvent.click(nextButton);

    await waitForLoadingToFinish(container);

    // Check the second page.
    checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(checkboxes[0]).toHaveTextContent(items[2].name);
    expect(checkboxes[0]).not.toBeChecked();
    nextButton = screen.getByRole("button", { name: /next/i });
    previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeDisabled();
    expect(previousButton).toBeEnabled();

    // Move back to first page.
    await userEvent.click(previousButton);

    await waitForLoadingToFinish(container);

    // Check the first page again.
    checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(2);
    [items[0], items[1]].forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
    });
    nextButton = screen.getByRole("button", { name: /next/i });
    previousButton = screen.getByRole("button", { name: /previous/i });
    expect(nextButton).toBeEnabled();
    expect(previousButton).toBeDisabled();
  });

  it("Adds pagination when > page size items are returned (with fetchCount provided)", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchTinyPage}
        fetchCount={() => Promise.resolve(3)}
        pageSize={1}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);

    const nextButton = screen.getByRole("button", { name: /next/i });
    const previousButton = screen.getByRole("button", { name: /previous/i });

    // Check the first page.
    let checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(checkboxes[0]).toHaveTextContent(items[0].name);
    expect(checkboxes[0]).not.toBeChecked();
    expect(nextButton).toBeEnabled();
    expect(previousButton).toBeDisabled();

    // Move to second page.
    await userEvent.click(nextButton);
    await waitForLoadingToFinish(container);

    // Check the second page.
    checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(checkboxes[0]).toHaveTextContent(items[1].name);
    expect(checkboxes[0]).not.toBeChecked();
    expect(nextButton).toBeEnabled();
    expect(previousButton).toBeEnabled();

    // Move to third page.
    await userEvent.click(nextButton);
    await waitForLoadingToFinish(container);

    // Check the third page.
    checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(1);
    expect(checkboxes[0]).toHaveTextContent(items[2].name);
    expect(checkboxes[0]).not.toBeChecked();
    expect(nextButton).toBeDisabled();
    expect(previousButton).toBeEnabled();
  });

  it("Allows for custom markup in item rows", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
        renderItemRow={(item) => <span>{item.favoriteIceCreamFlavor}</span>}
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.name);
      expect(checkboxes[index]).not.toBeChecked();
      expect(
        checkboxes[index].closest(".form-field")?.nextElementSibling
      ).toHaveTextContent(item.favoriteIceCreamFlavor);
    });
  });

  it("Allows for custom markup for item labels", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
        renderItemLabel={(item) => <span>{item.favoriteIceCreamFlavor}</span>}
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    items.forEach((item, index) => {
      expect(checkboxes[index]).toHaveTextContent(item.favoriteIceCreamFlavor);
      expect(checkboxes[index]).not.toBeChecked();
    });
  });

  it("Notifies the parent when an item is toggled and marks the item as dirty", async () => {
    const onToggleItem = jest.fn((item) => {
      return {
        ...item,
        checkMeBruh: !item.checkMeBruh,
      };
    });
    const paginatedListRef = createRef<IPaginatedListHandle<ITestItem>>();
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        ref={paginatedListRef}
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={onToggleItem}
        onUpdate={jest.fn()}
        isSelected="checkMeBruh"
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    expect(checkboxes[0]).toBeChecked();
    await userEvent.click(checkboxes[0]);
    expect(onToggleItem).toHaveBeenCalledWith(items[0]);

    // Check that the item is marked as dirty.
    await waitFor(() => {
      expect(paginatedListRef.current?.getDirtyItems()).toEqual([
        { ...items[0], checkMeBruh: false },
      ]);
    });
    expect(checkboxes[0]).not.toBeChecked();
  });

  it("Can update the set of dirty items when a change is made in custom markup", async () => {
    const paginatedListRef = createRef<IPaginatedListHandle<ITestItem>>();
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        ref={paginatedListRef}
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
        renderItemRow={(item, onChange) => (
          <button
            onClick={(e) => {
              e.stopPropagation();
              onChange({
                ...item,
                favoriteIceCreamFlavor: `${item.favoriteIceCreamFlavor} Pie`,
              });
            }}
          >
            Click me bruh
          </button>
        )}
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    userEvent.click(
      checkboxes[1].closest(".form-field")?.nextElementSibling as HTMLElement
    );
    // Check that the item is marked as dirty.
    await waitFor(() => {
      expect(paginatedListRef.current?.getDirtyItems()).toEqual([
        { ...items[1], favoriteIceCreamFlavor: "Dirt Pie" },
      ]);
    });
  });

  it("Notifies the parent when a change is made to the set of dirty items", async () => {
    const onUpdate = jest.fn();
    const paginatedListRef = createRef<IPaginatedListHandle<ITestItem>>();
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        ref={paginatedListRef}
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn((item) => item)}
        onUpdate={onUpdate}
        isSelected={jest.fn()}
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    userEvent.click(checkboxes[0]);
    await waitFor(() => {
      expect(onUpdate.mock.calls.length).toEqual(2);
      // Called on first render with no updated items.
      expect(onUpdate.mock.calls[0]).toEqual([[]]);
      // Called after toggle with the result of the toggle.
      expect(onUpdate.mock.calls[1]).toEqual([[items[0]]]);
    });
  });

  it("Allows for disabling the list", async () => {
    const { container } = renderWithSetup(
      <PaginatedList<ITestItem>
        fetchPage={fetchLargePage}
        pageSize={10}
        onToggleItem={jest.fn()}
        onUpdate={jest.fn()}
        isSelected={jest.fn()}
        disabled
      />
    );
    await waitForLoadingToFinish(container);
    const checkboxes = container.querySelectorAll("input[checkbox]");
    checkboxes.forEach((checkbox) => {
      expect(checkbox).toBeDisabled();
    });
  });
});

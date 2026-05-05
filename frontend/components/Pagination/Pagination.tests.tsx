import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import Pagination from "./Pagination";

describe("Pagination", () => {
  const defaultProps = {
    onNextPage: jest.fn(),
    onPrevPage: jest.fn(),
  };

  it("renders the pagination buttons and applies custom className when provided", () => {
    const customClass = "custom-pagination";
    render(<Pagination {...defaultProps} className={customClass} />);

    const prevButton = screen.getByText("Previous");
    const nextButton = screen.getByText("Next");

    expect(prevButton).toBeInTheDocument();
    expect(nextButton).toBeInTheDocument();

    // Find the parent container of the buttons
    const paginationContainer = prevButton.closest(`.pagination`);

    expect(paginationContainer).toHaveClass(customClass);
    expect(paginationContainer).toHaveClass("pagination"); // Check for the base class as well
  });

  it("disables the previous button when disablePrev is true and same for next button", () => {
    render(<Pagination {...defaultProps} disablePrev disableNext />);

    const prevButton = screen.getByRole("button", { name: /previous/i });
    const nextButton = screen.getByRole("button", { name: /next/i });

    expect(prevButton).toBeDisabled();
    expect(nextButton).toBeDisabled();
  });

  it("calls onPrevPage when the previous button is clicked and same for onNextPage", () => {
    const onPrevPageMock = jest.fn();
    const onNextPageMock = jest.fn();

    render(
      <Pagination
        {...defaultProps}
        onPrevPage={onPrevPageMock}
        onNextPage={onNextPageMock}
      />
    );

    const prevButton = screen.getByText("Previous");
    fireEvent.click(prevButton);
    const nextButton = screen.getByText("Next");
    fireEvent.click(nextButton);

    expect(onPrevPageMock).toHaveBeenCalledTimes(1);
    expect(onNextPageMock).toHaveBeenCalledTimes(1);
  });

  it("does not render pagination if hidePagination is true", () => {
    render(<Pagination {...defaultProps} hidePagination />);

    const prevButton = screen.queryByText("Previous");
    const nextButton = screen.queryByText("Next");

    expect(prevButton).not.toBeInTheDocument();
    expect(nextButton).not.toBeInTheDocument();
  });
});

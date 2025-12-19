import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import { renderWithSetup } from "test/test-utils";

import FeedListItem from "./FeedListItem";

describe("FeedListItem", () => {
  const defaultProps = {
    useFleetAvatar: true,
    createdAt: new Date("2024-01-01T12:00:00Z"),
  };

  it("renders the info icon when allowShowDetails is true", () => {
    render(
      <FeedListItem {...defaultProps} allowShowDetails>
        Test content
      </FeedListItem>
    );

    expect(screen.getByTestId("info-outline-icon")).toBeInTheDocument();
  });

  it("does not render the info icon when allowShowDetails is false", () => {
    render(
      <FeedListItem {...defaultProps} allowShowDetails={false}>
        Test content
      </FeedListItem>
    );

    expect(screen.queryByTestId("info-outline-icon")).not.toBeInTheDocument();
  });

  it("calls onClickFeedItem when allowShowDetails is true and the feed content is clicked", async () => {
    const onClickFeedItem = jest.fn();

    const { user } = renderWithSetup(
      <FeedListItem
        {...defaultProps}
        allowShowDetails
        onClickFeedItem={onClickFeedItem}
      >
        Test content
      </FeedListItem>
    );

    const detailsWrapper = screen.getByRole("button", {
      name: /Test content/i,
    });
    await user.click(detailsWrapper);

    expect(onClickFeedItem).toHaveBeenCalledTimes(1);
  });

  it("renders the createdAt date when passed in", () => {
    const createdAt = new Date("2024-01-01T12:00:00Z");
    render(
      <FeedListItem {...defaultProps} createdAt={createdAt}>
        Test content
      </FeedListItem>
    );

    // The dateAgo function will render something like "X days ago" or similar
    // We can check that some date-related text is rendered
    const dateElement = screen.getByText(/ago/i);
    expect(dateElement).toBeInTheDocument();
  });

  it("renders the close icon when allowCancel is true", () => {
    render(
      <FeedListItem {...defaultProps} allowCancel>
        Test content
      </FeedListItem>
    );

    expect(screen.getByTestId("close-icon")).toBeInTheDocument();
  });

  it("does not render the close icon when allowCancel is false", () => {
    render(
      <FeedListItem {...defaultProps} allowCancel={false}>
        Test content
      </FeedListItem>
    );

    expect(screen.queryByTestId("close-icon")).not.toBeInTheDocument();
  });

  it("calls onClickCancel when the close icon is clicked", async () => {
    const onClickCancel = jest.fn();

    render(
      <FeedListItem {...defaultProps} allowCancel onClickCancel={onClickCancel}>
        Test content
      </FeedListItem>
    );

    // there is some weirdness with userEvent.click on the cancel icon
    // so using fireEvent.click for this test. I think it has to do with
    // the icon not being shown until the user hovers over the button but we
    // were not able to fix this so we use fireEvent.click instead.
    const cancelIcon = screen.getByRole("button", { name: "cancel action" });
    await fireEvent.click(cancelIcon);

    expect(onClickCancel).toHaveBeenCalledTimes(1);
  });

  it("disables the close button when disableCancel is true", () => {
    render(
      <FeedListItem {...defaultProps} allowCancel disableCancel>
        Test content
      </FeedListItem>
    );

    const cancelIcon = screen.getByRole("button", { name: "cancel action" });
    expect(cancelIcon).toBeDisabled();
  });
});

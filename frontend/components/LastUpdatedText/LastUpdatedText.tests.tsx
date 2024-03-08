import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import LastUpdatedText from ".";

describe("Last updated text", () => {
  it("renders updated text", () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);
    const twoDaysAgo = currentDate.toISOString();

    render(
      <LastUpdatedText whatToRetrieve="software" lastUpdatedAt={twoDaysAgo} />
    );

    const text = screen.getByText("Updated 2 days ago");

    expect(text).toBeInTheDocument();
  });
  it("renders never if missing timestamp", () => {
    render(<LastUpdatedText whatToRetrieve="software" />);

    const text = screen.getByText("Updated never");

    expect(text).toBeInTheDocument();
  });

  it("renders tooltip on hover", async () => {
    render(<LastUpdatedText whatToRetrieve="software" />);

    await fireEvent.mouseEnter(screen.getByText("Updated never"));

    expect(screen.getByText(/to retrieve software/i)).toBeInTheDocument();
  });
});

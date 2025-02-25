import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";

import LastUpdatedHostCount from ".";

describe("Last updated host count", () => {
  it("renders host count and updated text", () => {
    const currentDate = new Date();
    currentDate.setDate(currentDate.getDate() - 2);
    const twoDaysAgo = currentDate.toISOString();

    render(<LastUpdatedHostCount hostCount={40} lastUpdatedAt={twoDaysAgo} />);

    const hostCount = screen.getByText(/40/i);
    const updateText = screen.getByText("Updated 2 days ago");

    expect(hostCount).toBeInTheDocument();
    expect(updateText).toBeInTheDocument();
  });
  it("renders never if missing timestamp", () => {
    render(<LastUpdatedHostCount />);

    const text = screen.getByText("Updated never");

    expect(text).toBeInTheDocument();
  });

  it("renders tooltip on hover", async () => {
    render(<LastUpdatedHostCount hostCount={0} />);

    await fireEvent.mouseEnter(screen.getByText("Updated never"));

    expect(
      screen.getByText(/last time host data was updated/i)
    ).toBeInTheDocument();
  });
});

import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import { renderWithSetup } from "test/test-utils";

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

  it("renders 'Updated never' if lastUpdatedAt is explicitly null", () => {
    render(<LastUpdatedHostCount lastUpdatedAt={null} />);
    expect(screen.getByText("Updated never")).toBeInTheDocument();
  });

  it("does not render updated text if lastUpdatedAt is undefined", () => {
    render(<LastUpdatedHostCount />);
    expect(screen.queryByText(/Updated/i)).not.toBeInTheDocument();
  });

  it("renders tooltip on hover when 'Updated never'", async () => {
    const { user } = renderWithSetup(
      <LastUpdatedHostCount hostCount={0} lastUpdatedAt={null} />
    );
    await user.hover(screen.getByText("Updated never"));
    await waitFor(() => {
      expect(
        screen.getByText(/last time host data was updated/i)
      ).toBeInTheDocument();
    });
  });
});

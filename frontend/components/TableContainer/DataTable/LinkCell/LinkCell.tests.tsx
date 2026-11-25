import { screen, waitFor } from "@testing-library/react";
import React from "react";

import PATHS from "router/paths";
import { renderWithSetup } from "test/test-utils";

import LinkCell from "./LinkCell";

const VALUE = "40 hosts";
describe("Link cell", () => {
  it("renders text", async () => {
    renderWithSetup(<LinkCell value={VALUE} path={PATHS.MANAGE_HOSTS} />);

    expect(screen.getByText("40 hosts")).toBeInTheDocument();
    // Note: Testing react-router Link would require Router or MemoryRouter wrapper which is app level
  });

  describe("truncation tooltip", () => {
    // jsdom has no layout, so report the text as wider than its box to make it count as truncated.
    beforeEach(() => {
      jest
        .spyOn(HTMLElement.prototype, "scrollWidth", "get")
        .mockReturnValue(500);
    });
    afterEach(() => {
      jest.restoreAllMocks();
    });

    const renderTruncatedLinkCell = (onClick = jest.fn()) =>
      renderWithSetup(
        <LinkCell
          value={VALUE}
          path={PATHS.MANAGE_HOSTS}
          customOnClick={onClick}
          tooltipTruncate
        />
      );

    it("opens outside the link, so clicking it does not follow the link", async () => {
      const onClick = jest.fn();
      const { container, user } = renderTruncatedLinkCell(onClick);

      await user.hover(screen.getByText(VALUE));
      const tooltip = await screen.findByRole("tooltip");

      // The tooltip overlaps the row above, so a click on it must not navigate to this row.
      expect(container.querySelector("a")).not.toContainElement(tooltip);
      await user.click(tooltip);
      expect(onClick).not.toHaveBeenCalled();
    });

    it("stays open while hovered and closes on Escape", async () => {
      const { user } = renderTruncatedLinkCell();

      await user.hover(screen.getByText(VALUE));
      const tooltip = await screen.findByRole("tooltip");

      await user.hover(tooltip);
      // Wait so a tooltip that closes when the pointer leaves the link would be gone.
      await new Promise((resolve) => setTimeout(resolve, 300));
      expect(screen.getByRole("tooltip")).toBeInTheDocument();

      await user.keyboard("{Escape}");
      await waitFor(() => {
        expect(screen.queryByRole("tooltip")).not.toBeInTheDocument();
      });
    });
  });
});

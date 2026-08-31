import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import InstallAllInCategoryModal from "./InstallAllInCategoryModal";

describe("InstallAllInCategoryModal", () => {
  it("renders the title and the confirm button (both labeled 'Install all')", () => {
    render(
      <InstallAllInCategoryModal
        count={3}
        onConfirm={jest.fn()}
        onExit={jest.fn()}
      />
    );
    // Both the modal heading and the confirm button have the same text.
    expect(screen.getAllByText("Install all")).toHaveLength(2);
  });

  it("pluralizes the count for >1", () => {
    render(
      <InstallAllInCategoryModal
        count={5}
        onConfirm={jest.fn()}
        onExit={jest.fn()}
      />
    );
    expect(
      screen.getByText(/5 new apps will be installed/i)
    ).toBeInTheDocument();
  });

  it("uses singular wording for count=1", () => {
    render(
      <InstallAllInCategoryModal
        count={1}
        onConfirm={jest.fn()}
        onExit={jest.fn()}
      />
    );
    expect(
      screen.getByText(/1 new app will be installed/i)
    ).toBeInTheDocument();
  });

  it("calls onConfirm when the confirm button is clicked", async () => {
    const onConfirm = jest.fn();
    const user = userEvent.setup();
    render(
      <InstallAllInCategoryModal
        count={3}
        onConfirm={onConfirm}
        onExit={jest.fn()}
      />
    );

    await user.click(screen.getByRole("button", { name: /^Install all$/i }));

    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onExit when Cancel is clicked", async () => {
    const onExit = jest.fn();
    const user = userEvent.setup();
    render(
      <InstallAllInCategoryModal
        count={3}
        onConfirm={jest.fn()}
        onExit={onExit}
      />
    );

    await user.click(screen.getByRole("button", { name: /Cancel/i }));

    expect(onExit).toHaveBeenCalledTimes(1);
  });

  it("disables the Cancel button while submitting", () => {
    render(
      <InstallAllInCategoryModal
        count={3}
        isSubmitting
        onConfirm={jest.fn()}
        onExit={jest.fn()}
      />
    );
    expect(screen.getByRole("button", { name: /Cancel/i })).toBeDisabled();
  });
});

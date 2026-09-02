/**
 * Tests ToastCard's expandable detail panel: which payloads are worth
 * revealing and which are suppressed.
 */
import React from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import ToastCard from "./ToastCard";

jest.mock("sonner", () => ({
  toast: {
    custom: jest.fn(),
    dismiss: jest.fn(),
  },
  Toaster: () => null,
}));

const EXPAND_LABEL = "Expand error details";

const renderCard = (detail?: unknown) =>
  render(
    <ToastCard
      variant="error"
      message="Couldn't upload. Please try again."
      detail={detail}
      toastId="test-id"
    />
  );

describe("ToastCard", () => {
  it("hides the details toggle when an Error is passed as the payload", () => {
    // Regression test for #50846. An Error's own properties are
    // non-enumerable, so JSON.stringify returns "{}" without throwing and
    // the panel used to open on an empty object.
    renderCard(new Error("unsupported file extension: dmg"));

    expect(
      screen.queryByRole("button", { name: EXPAND_LABEL })
    ).not.toBeInTheDocument();
  });

  it.each([
    ["an empty object", {}],
    ["an empty array", []],
    ["null", null],
    ["an empty string", ""],
    // These have no JSON representation, so JSON.stringify returns undefined
    // rather than throwing.
    ["a function", () => "noop"],
    ["a symbol", Symbol("token")],
  ])("hides the details toggle for %s", (_label, detail) => {
    renderCard(detail);

    expect(
      screen.queryByRole("button", { name: EXPAND_LABEL })
    ).not.toBeInTheDocument();
  });

  it("hides the details toggle when no payload is provided", () => {
    renderCard();

    expect(
      screen.queryByRole("button", { name: EXPAND_LABEL })
    ).not.toBeInTheDocument();
  });

  it("shows the details toggle when the payload has content", () => {
    renderCard({ message: "unsupported file extension: dmg" });

    expect(
      screen.getByRole("button", { name: EXPAND_LABEL })
    ).toBeInTheDocument();
  });

  it("reveals the payload when the toggle is clicked", async () => {
    const user = userEvent.setup();
    renderCard({ status: 422 });

    await user.click(screen.getByRole("button", { name: EXPAND_LABEL }));

    expect(screen.getByRole("region", { name: "Error details" })).toBeVisible();
    expect(screen.getByText(/422/)).toBeInTheDocument();
  });
});
